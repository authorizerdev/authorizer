package totp

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// recoveryStore is a correct backend in miniature: the compare of oldCodes and
// the write happen under one lock, which is what every real implementation gets
// from its database (row lock, LWT, ConditionExpression, CAS). It exists so the
// retry loop in ValidateRecoveryCode can be tested for the behaviours a real
// database only produces under luck — a lost race, a run of lost races, a
// storage fault — without needing six databases running.
type recoveryStore struct {
	storage.Provider

	mu    sync.Mutex
	row   *schemas.Authenticator
	reads int

	// beforeConsume runs outside the lock, so a test can commit an interfering
	// write in the window between the caller's read and its swap.
	beforeConsume func()
	// consumeErr, when set, makes every swap fail as a storage fault.
	consumeErr error
	// loseFirstN refuses that many swaps before behaving normally, standing in
	// for other requests committing to the row.
	loseFirstN int
}

func (s *recoveryStore) GetAuthenticatorDetailsByUserId(_ context.Context, _, _ string) (*schemas.Authenticator, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads++
	// Hand back a copy: a caller must never be able to mutate the stored row
	// by writing through the struct it was given.
	cp := *s.row
	codes := refs.StringValue(s.row.RecoveryCodes)
	cp.RecoveryCodes = &codes
	return &cp, nil
}

func (s *recoveryStore) ConsumeAuthenticatorRecoveryCode(_ context.Context, id, oldCodes, newCodes string) (bool, error) {
	if s.beforeConsume != nil {
		s.beforeConsume()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.consumeErr != nil {
		return false, s.consumeErr
	}
	if s.loseFirstN > 0 {
		s.loseFirstN--
		return false, nil
	}
	if id != s.row.ID || refs.StringValue(s.row.RecoveryCodes) != oldCodes {
		return false, nil
	}
	s.row.RecoveryCodes = &newCodes
	return true, nil
}

func (s *recoveryStore) codes(t *testing.T) map[string]bool {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]bool{}
	require.NoError(t, json.Unmarshal([]byte(refs.StringValue(s.row.RecoveryCodes)), &out))
	return out
}

// newRecoveryProvider builds a provider over a store holding n fresh hashed
// recovery codes, and returns the plaintext codes.
func newRecoveryProvider(t *testing.T, n int) (*provider, *recoveryStore, []string) {
	t.Helper()
	plain := make([]string, n)
	stored := map[string]bool{}
	for i := range plain {
		plain[i] = uuid.NewString()
		stored[crypto.HashRecoveryCode(plain[i])] = false
	}
	blob, err := json.Marshal(stored)
	require.NoError(t, err)

	store := &recoveryStore{row: &schemas.Authenticator{
		ID:            uuid.NewString(),
		UserID:        "user-1",
		Method:        "totp",
		RecoveryCodes: refs.NewStringRef(string(blob)),
	}}

	l := zerolog.Nop()
	p, err := NewProvider(&Dependencies{
		Log:             &l,
		StorageProvider: store,
		EncryptionKey:   "test-key",
	})
	require.NoError(t, err)
	return p, store, plain
}

// TestValidateRecoveryCodeIsSingleUseUnderConcurrency is the regression guard
// for the double-spend. Read-decide-write let every racer see the code
// unconsumed and return true; the swap has to be what enforces "once".
func TestValidateRecoveryCodeIsSingleUseUnderConcurrency(t *testing.T) {
	const racers = 16
	for round := 0; round < 50; round++ {
		p, store, plain := newRecoveryProvider(t, 10)

		var start, done sync.WaitGroup
		var mu sync.Mutex
		accepted := 0
		start.Add(1)
		for i := 0; i < racers; i++ {
			done.Add(1)
			go func() {
				defer done.Done()
				start.Wait()
				ok, err := p.ValidateRecoveryCode(context.Background(), plain[0], "user-1")
				assert.NoError(t, err)
				if ok {
					mu.Lock()
					accepted++
					mu.Unlock()
				}
			}()
		}
		start.Done()
		done.Wait()

		require.Equal(t, 1, accepted, "round %d: one code, one redemption", round)
		assert.True(t, store.codes(t)[crypto.HashRecoveryCode(plain[0])], "the redeemed code must be marked consumed")
	}
}

// TestValidateRecoveryCodeRetriesAfterALostRace pins the recovery half of the
// contract: losing the swap is not a rejection. Another request writing to the
// row does not make THIS caller's code invalid, so the loop re-reads and
// re-applies rather than answering false.
func TestValidateRecoveryCodeRetriesAfterALostRace(t *testing.T) {
	p, store, plain := newRecoveryProvider(t, 10)
	store.loseFirstN = recoveryCodeConsumeAttempts - 1

	ok, err := p.ValidateRecoveryCode(context.Background(), plain[0], "user-1")
	require.NoError(t, err)
	assert.True(t, ok, "a code that is still unconsumed must be accepted despite lost races")
	assert.Equal(t, recoveryCodeConsumeAttempts, store.reads, "each retry must re-read rather than reuse a stale blob")
	assert.True(t, store.codes(t)[crypto.HashRecoveryCode(plain[0])])
}

// TestValidateRecoveryCodeLosesToAConcurrentRedemptionOfTheSameCode drives the
// exact interleaving the fix targets: the code is spent by someone else in the
// window between this caller's read and its swap. The retry must re-read, find
// it consumed, and reject — cleanly, with no error.
func TestValidateRecoveryCodeLosesToAConcurrentRedemptionOfTheSameCode(t *testing.T) {
	p, store, plain := newRecoveryProvider(t, 10)

	var once sync.Once
	store.beforeConsume = func() {
		once.Do(func() {
			// Someone else redeems the same code first.
			store.mu.Lock()
			codes := map[string]bool{}
			_ = json.Unmarshal([]byte(refs.StringValue(store.row.RecoveryCodes)), &codes)
			codes[crypto.HashRecoveryCode(plain[0])] = true
			blob, _ := json.Marshal(codes)
			store.row.RecoveryCodes = refs.NewStringRef(string(blob))
			store.mu.Unlock()
		})
	}

	ok, err := p.ValidateRecoveryCode(context.Background(), plain[0], "user-1")
	require.NoError(t, err, "losing the race is a rejection, not a fault")
	assert.False(t, ok, "a code spent by another request must not validate here too")
}

// TestValidateRecoveryCodeReportsSustainedContentionAsAFault pins the direction
// the loop fails in. Exhausting the retries means the WRITE never landed — the
// code may well still be valid — so answering false would spend a user's
// recovery credential on a database problem and tell them their input was
// wrong. It must be an error, and the bool must stay false.
func TestValidateRecoveryCodeReportsSustainedContentionAsAFault(t *testing.T) {
	p, store, plain := newRecoveryProvider(t, 10)
	store.loseFirstN = recoveryCodeConsumeAttempts

	ok, err := p.ValidateRecoveryCode(context.Background(), plain[0], "user-1")
	require.Error(t, err, "exhausted retries must not be reported as an invalid code")
	assert.False(t, ok)
	assert.False(t, store.codes(t)[crypto.HashRecoveryCode(plain[0])], "an unconsumed code must remain spendable")
}

// TestValidateRecoveryCodePropagatesStorageErrors keeps a database outage
// distinguishable from a wrong code, per the not-found contract in AGENTS.md.
func TestValidateRecoveryCodePropagatesStorageErrors(t *testing.T) {
	p, store, plain := newRecoveryProvider(t, 10)
	boom := errors.New("database is on fire")
	store.consumeErr = boom

	ok, err := p.ValidateRecoveryCode(context.Background(), plain[0], "user-1")
	require.ErrorIs(t, err, boom)
	assert.False(t, ok)
}

// TestValidateRecoveryCodeRejectionsDoNotWrite covers the two rejections that
// must never reach the database at all: an unknown code and an already-spent
// one. Both are failed attempts, not faults.
func TestValidateRecoveryCodeRejectionsDoNotWrite(t *testing.T) {
	p, store, plain := newRecoveryProvider(t, 10)
	// Make any swap that does happen loudly wrong.
	store.consumeErr = errors.New("no write should have been attempted")

	ok, err := p.ValidateRecoveryCode(context.Background(), uuid.NewString(), "user-1")
	require.NoError(t, err, "an unissued code is a failed attempt, not an error")
	assert.False(t, ok)

	store.consumeErr = nil
	ok, err = p.ValidateRecoveryCode(context.Background(), plain[0], "user-1")
	require.NoError(t, err)
	require.True(t, ok)

	store.consumeErr = errors.New("no write should have been attempted")
	ok, err = p.ValidateRecoveryCode(context.Background(), plain[0], "user-1")
	require.NoError(t, err, "a spent code is a failed attempt, not an error")
	assert.False(t, ok)
}

// TestValidateRecoveryCodeConsumesOnlyTheMatchedCode pins that the swap offered
// to storage differs from the stored blob in exactly one key. The whole blob is
// rewritten on every redemption, so a bug here silently burns or revives the
// other nine codes.
func TestValidateRecoveryCodeConsumesOnlyTheMatchedCode(t *testing.T) {
	p, store, plain := newRecoveryProvider(t, 10)

	ok, err := p.ValidateRecoveryCode(context.Background(), plain[3], "user-1")
	require.NoError(t, err)
	require.True(t, ok)

	after := store.codes(t)
	require.Len(t, after, 10, "no code may be added or dropped by a redemption")
	for i, code := range plain {
		assert.Equal(t, i == 3, after[crypto.HashRecoveryCode(code)],
			"only the redeemed code may change state (index %d)", i)
	}
}

// TestValidateRecoveryCodeConsumesLegacyPlaintextCodes keeps the rolling-upgrade
// fallback working through the new swap: a row written by a pre-hashing release
// stores the code itself as the key, and the compare-and-swap must be built from
// that same blob or the write never lands.
func TestValidateRecoveryCodeConsumesLegacyPlaintextCodes(t *testing.T) {
	legacyCode := uuid.NewString()
	blob, err := json.Marshal(map[string]bool{legacyCode: false})
	require.NoError(t, err)

	store := &recoveryStore{row: &schemas.Authenticator{
		ID:            uuid.NewString(),
		UserID:        "user-1",
		Method:        "totp",
		RecoveryCodes: refs.NewStringRef(string(blob)),
	}}
	l := zerolog.Nop()
	p, err := NewProvider(&Dependencies{Log: &l, StorageProvider: store, EncryptionKey: "test-key"})
	require.NoError(t, err)

	ok, err := p.ValidateRecoveryCode(context.Background(), legacyCode, "user-1")
	require.NoError(t, err)
	assert.True(t, ok, "a legacy plaintext code must still validate during a rolling upgrade")
	assert.True(t, store.codes(t)[legacyCode], "and be consumed under its plaintext key")

	ok, err = p.ValidateRecoveryCode(context.Background(), legacyCode, "user-1")
	require.NoError(t, err)
	assert.False(t, ok, "a legacy code is single-use too")
}

// TestValidateRecoveryCodeRejectsACorruptBlob pins that an unparseable
// recovery-code column is a fault, not a silent "wrong code" — the latter would
// hide the corruption behind a plausible-looking login failure.
func TestValidateRecoveryCodeRejectsACorruptBlob(t *testing.T) {
	store := &recoveryStore{row: &schemas.Authenticator{
		ID:            uuid.NewString(),
		UserID:        "user-1",
		Method:        "totp",
		RecoveryCodes: refs.NewStringRef("not json"),
	}}
	l := zerolog.Nop()
	p, err := NewProvider(&Dependencies{Log: &l, StorageProvider: store, EncryptionKey: "test-key"})
	require.NoError(t, err)

	ok, err := p.ValidateRecoveryCode(context.Background(), uuid.NewString(), "user-1")
	assert.Error(t, err)
	assert.False(t, ok)
}

package totp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	otptotp "github.com/pquerna/otp/totp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// disjointWriteStore models what every real backend does with the two writes to
// an authenticator row, so the interaction between them can be tested without
// six databases:
//
//   - UpdateAuthenticator replaces the WHOLE row from the caller's struct,
//     recovery codes included (SQL Save, Mongo $set of the full struct, and so
//     on). If Validate ever routes through it, a stale blob goes back.
//   - UpdateAuthenticatorSecretAndVerifiedAt touches only its two columns.
//
// beforeWrite fires just before either write lands, which is where the
// interfering redemption is injected.
type disjointWriteStore struct {
	*recoveryStore

	beforeWrite             func()
	updateAuthenticatorHits int
	narrowUpdateHits        int
}

func (s *disjointWriteStore) UpdateAuthenticator(_ context.Context, a *schemas.Authenticator) (*schemas.Authenticator, error) {
	if s.beforeWrite != nil {
		s.beforeWrite()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateAuthenticatorHits++
	cp := *a
	s.row = &cp
	return a, nil
}

func (s *disjointWriteStore) UpdateAuthenticatorSecretAndVerifiedAt(_ context.Context, id, secret string, verifiedAt int64) error {
	if s.beforeWrite != nil {
		s.beforeWrite()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.narrowUpdateHits++
	if id != s.row.ID {
		return nil
	}
	s.row.Secret = secret
	s.row.VerifiedAt = &verifiedAt
	s.row.UpdatedAt = time.Now().Unix()
	return nil
}

// TestValidateDoesNotResurrectAConsumedRecoveryCode is the regression guard for
// the residual left open by the single-use fix.
//
// Validate reads the authenticator row, then spends real time on it — decrypt,
// TOTP check, replay reservation — and only then writes back VerifiedAt. While
// it did that through the whole-row UpdateAuthenticator, the recovery-code blob
// it had read at the start went back to the server too, restoring a code that a
// concurrent redemption had consumed in the meantime. One code, spendable
// again, with no error anywhere.
//
// The interfering redemption is injected exactly in that window, so the failure
// is deterministic rather than a race the test has to win.
func TestValidateDoesNotResurrectAConsumedRecoveryCode(t *testing.T) {
	key, err := otptotp.Generate(otptotp.GenerateOpts{Issuer: "authorizer", AccountName: "user@authorizer.dev"})
	require.NoError(t, err)
	plainSecret := key.Secret()

	const encryptionKey = "test-key"
	encryptedSecret, err := crypto.EncryptTOTPSecret(plainSecret, encryptionKey)
	require.NoError(t, err)

	// Ten fresh codes, none consumed. VerifiedAt is nil so Validate takes its
	// write path (first-ever validation).
	plain := make([]string, totalRecoveryCodes)
	stored := map[string]bool{}
	for i := range plain {
		plain[i] = uuid.NewString()
		stored[crypto.HashRecoveryCode(plain[i])] = false
	}
	blob, err := json.Marshal(stored)
	require.NoError(t, err)

	base := &recoveryStore{row: &schemas.Authenticator{
		ID:            uuid.NewString(),
		UserID:        "user-1",
		Method:        "totp",
		Secret:        encryptedSecret,
		RecoveryCodes: refs.NewStringRef(string(blob)),
	}}
	store := &disjointWriteStore{recoveryStore: base}

	l := zerolog.Nop()
	p, err := NewProvider(&Dependencies{
		Log:             &l,
		StorageProvider: store,
		EncryptionKey:   encryptionKey,
	})
	require.NoError(t, err)

	// Someone redeems a recovery code in the window between Validate's read and
	// its write. This is a real redemption through the real code path, so what
	// it writes is exactly what a concurrent request would have written.
	var interfered bool
	store.beforeWrite = func() {
		if interfered {
			return
		}
		interfered = true
		ok, err := p.ValidateRecoveryCode(context.Background(), plain[0], "user-1")
		assert.NoError(t, err)
		assert.True(t, ok, "the interfering redemption must itself succeed")
	}

	passcode, err := otptotp.GenerateCode(plainSecret, time.Now())
	require.NoError(t, err)

	ok, err := p.Validate(context.Background(), passcode, "user-1")
	require.NoError(t, err)
	require.True(t, ok, "a valid passcode must still authenticate")
	require.True(t, interfered, "the test is meaningless unless Validate wrote at all")

	// The claim under test: the redeemed code is still spent.
	assert.True(t, store.codes(t)[crypto.HashRecoveryCode(plain[0])],
		"a recovery code consumed during Validate must not be restored to unconsumed")

	// Structural guarantee behind it — Validate must never reach the whole-row
	// write. Asserting the outcome alone would pass again the moment someone
	// swaps the call back.
	assert.Zero(t, store.updateAuthenticatorHits,
		"Validate must not write the whole row; recovery_codes is not its column")
	assert.Equal(t, 1, store.narrowUpdateHits, "Validate must write through the narrow update")

	// And Validate's own two columns did land.
	store.mu.Lock()
	defer store.mu.Unlock()
	assert.NotNil(t, store.row.VerifiedAt, "first successful validation must record VerifiedAt")
	assert.Equal(t, encryptedSecret, store.row.Secret, "an already-encrypted secret must be written back unchanged")

	// The nine untouched codes are still unconsumed — the narrow write neither
	// restored the spent one nor burned the rest.
	consumed := map[string]bool{}
	require.NoError(t, json.Unmarshal([]byte(refs.StringValue(store.row.RecoveryCodes)), &consumed))
	require.Len(t, consumed, totalRecoveryCodes)
	for i, code := range plain {
		assert.Equal(t, i == 0, consumed[crypto.HashRecoveryCode(code)], "code %d", i)
	}
}

// TestValidateMigratesALegacySecretWithoutTouchingRecoveryCodes covers the other
// branch that writes: a pre-encryption plaintext secret is re-encrypted in
// place. It goes through the same narrow write, so the migration must not carry
// a stale recovery-code blob either.
func TestValidateMigratesALegacySecretWithoutTouchingRecoveryCodes(t *testing.T) {
	key, err := otptotp.Generate(otptotp.GenerateOpts{Issuer: "authorizer", AccountName: "user@authorizer.dev"})
	require.NoError(t, err)
	plainSecret := key.Secret()

	plain := uuid.NewString()
	blob, err := json.Marshal(map[string]bool{crypto.HashRecoveryCode(plain): false})
	require.NoError(t, err)

	verifiedAt := time.Now().Add(-time.Hour).Unix()
	base := &recoveryStore{row: &schemas.Authenticator{
		ID:     uuid.NewString(),
		UserID: "user-1",
		Method: "totp",
		// Legacy row: the secret is stored as raw base32, not enc:v1:.
		Secret:        plainSecret,
		RecoveryCodes: refs.NewStringRef(string(blob)),
		VerifiedAt:    &verifiedAt,
	}}
	store := &disjointWriteStore{recoveryStore: base}

	l := zerolog.Nop()
	p, err := NewProvider(&Dependencies{
		Log:             &l,
		StorageProvider: store,
		EncryptionKey:   "test-key",
	})
	require.NoError(t, err)

	var interfered bool
	store.beforeWrite = func() {
		if interfered {
			return
		}
		interfered = true
		ok, err := p.ValidateRecoveryCode(context.Background(), plain, "user-1")
		assert.NoError(t, err)
		assert.True(t, ok)
	}

	passcode, err := otptotp.GenerateCode(plainSecret, time.Now())
	require.NoError(t, err)

	ok, err := p.Validate(context.Background(), passcode, "user-1")
	require.NoError(t, err)
	require.True(t, ok)
	require.True(t, interfered, "the migration branch must have written")

	assert.Zero(t, store.updateAuthenticatorHits)
	assert.True(t, store.codes(t)[crypto.HashRecoveryCode(plain)],
		"a code consumed during a lazy secret migration must stay consumed")

	store.mu.Lock()
	defer store.mu.Unlock()
	assert.NotEqual(t, plainSecret, store.row.Secret, "the legacy secret must have been re-encrypted")
	assert.Equal(t, verifiedAt, refs.Int64Value(store.row.VerifiedAt),
		"an already-verified row must keep its original VerifiedAt")
}

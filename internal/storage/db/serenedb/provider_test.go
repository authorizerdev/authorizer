package serenedb

import (
	"context"
	"net"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const testDBURL = "postgres://postgres:postgres@localhost:7890/postgres"

// requireSereneDB skips unless the SereneDB container started by
// `make test-serenedb` is in play. Gated on TEST_DBS first, the way the SQL
// migration tests are, so `make test` (TEST_DBS=sqlite) stays Docker-free even
// on a machine that happens to have something bound to :7890.
func requireSereneDB(t *testing.T) {
	t.Helper()
	if !slices.Contains(strings.Split(os.Getenv("TEST_DBS"), ","), constants.DbTypeSereneDB) {
		t.Skip("set TEST_DBS=serenedb (make test-serenedb) to run SereneDB tests")
	}
	conn, err := net.DialTimeout("tcp", "localhost:7890", 2*time.Second)
	if err != nil {
		t.Skipf("skipping SereneDB tests: not reachable on localhost:7890: %v", err)
	}
	_ = conn.Close()
}

// newTestProvider connects to the shared test database.
func newTestProvider(t *testing.T) *provider {
	t.Helper()
	requireSereneDB(t)
	return newProviderAt(t, testDBURL)
}

func newProviderAt(t *testing.T, url string) *provider {
	t.Helper()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	p, err := NewProvider(&config.Config{
		DatabaseType: constants.DbTypeSereneDB,
		DatabaseURL:  url,
		DatabaseName: "authorizer_test",
	}, &Dependencies{Log: &logger})
	require.NoError(t, err)
	require.NotNil(t, p)
	return p
}

// TestAddVerificationRequestUpsert covers the SQL provider's
// ON CONFLICT (email, identifier) upsert. SereneDB before 26.11 rejected a
// CREATE UNIQUE INDEX as a conflict target, which needed a dedicated override;
// this pins that the stock upsert now replaces the row instead of erroring or
// leaving a duplicate.
func TestAddVerificationRequestUpsert(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close() //nolint:errcheck
	ctx := context.Background()

	email := uuid.New().String() + "@authorizer.dev"
	identifier := "basic_auth_signup"
	// Tokens must be unique per run: the table is not truncated between runs
	// and GetVerificationRequestByToken looks them up globally.
	tokenOne := "token-1-" + uuid.New().String()
	tokenTwo := "token-2-" + uuid.New().String()

	first, err := p.AddVerificationRequest(ctx, &schemas.VerificationRequest{
		Email:      email,
		Identifier: identifier,
		Token:      tokenOne,
		ExpiresAt:  time.Now().Add(time.Hour).Unix(),
		Nonce:      "nonce-1",
	})
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := p.AddVerificationRequest(ctx, &schemas.VerificationRequest{
		Email:      email,
		Identifier: identifier,
		Token:      tokenTwo,
		ExpiresAt:  time.Now().Add(2 * time.Hour).Unix(),
		Nonce:      "nonce-2",
	})
	require.NoError(t, err, "re-requesting verification must upsert, not fail on the unique index")
	require.NotNil(t, second)

	// The live token is the new one, and the old one is gone — one row, updated.
	got, err := p.GetVerificationRequestByToken(ctx, tokenTwo)
	require.NoError(t, err)
	assert.Equal(t, email, got.Email)
	assert.Equal(t, "nonce-2", got.Nonce)

	_, err = p.GetVerificationRequestByToken(ctx, tokenOne)
	assert.Error(t, err, "the superseded token must no longer resolve")

	var count int64
	require.NoError(t, p.DB().WithContext(ctx).Model(&schemas.VerificationRequest{}).
		Where("email = ? AND identifier = ?", email, identifier).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

// TestAddAuthenticatorConcurrentEnrollment covers the (user_id, method)
// upsert. The check-then-insert in AddAuthenticator has a race; on PostgreSQL
// ON CONFLICT closes it. Concurrent enrollment must still leave exactly one
// row, or GetAuthenticatorDetailsByUserId's First() returns an arbitrary one
// and MFA fails intermittently.
func TestAddAuthenticatorConcurrentEnrollment(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close() //nolint:errcheck
	ctx := context.Background()

	user, err := p.AddUser(ctx, &schemas.User{
		Email:         refs.NewStringRef(uuid.New().String() + "@authorizer.dev"),
		SignupMethods: constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)

	const goroutines = 8
	var wg sync.WaitGroup
	errs := make([]error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = p.AddAuthenticator(ctx, &schemas.Authenticator{
				UserID: user.ID,
				Method: constants.EnvKeyTOTPAuthenticator,
				Secret: "secret",
			})
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		assert.NoError(t, err, "concurrent enrollment %d must not surface the unique violation", i)
	}

	var count int64
	require.NoError(t, p.DB().WithContext(ctx).Model(&schemas.Authenticator{}).
		Where("user_id = ? AND method = ?", user.ID, constants.EnvKeyTOTPAuthenticator).
		Count(&count).Error)
	assert.Equal(t, int64(1), count, "concurrent enrollment must not duplicate the authenticator")
}

// TestAuthenticatorUniqueIndexEnforced is the backstop for the create-only
// migrator. AddAuthenticator's ON CONFLICT only works because the
// (user_id, method) unique index exists and raises 23505 — if the migrator ever
// stopped creating it, the concurrency test above would go quietly flaky
// instead of failing. This writes the duplicate row directly, bypassing the
// pre-check, and asserts the index rejects it.
func TestAuthenticatorUniqueIndexEnforced(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close() //nolint:errcheck
	ctx := context.Background()

	userID := uuid.New().String()
	first := &schemas.Authenticator{
		ID: uuid.New().String(), Key: uuid.New().String(),
		UserID: userID, Method: constants.EnvKeyTOTPAuthenticator, Secret: "one",
	}
	require.NoError(t, p.DB().WithContext(ctx).Create(first).Error)

	second := &schemas.Authenticator{
		ID: uuid.New().String(), Key: uuid.New().String(),
		UserID: userID, Method: constants.EnvKeyTOTPAuthenticator, Secret: "two",
	}
	err := p.DB().WithContext(ctx).Create(second).Error
	require.Error(t, err, "a second enrollment for the same (user_id, method) must be rejected")
	assert.True(t, uniqueViolation(err), "expected SQLSTATE 23505, got %v", err)
}

// TestDeleteUserCascadeStaysAtomic guards the one place this provider's GORM
// config differs from the SQL one. SkipDefaultTransaction removes the implicit
// transaction GORM wraps around each single write — required, because that
// transaction routes every write through BeginTx and past the pool's conflict
// retry. Explicit Transaction() blocks are supposed to be unaffected, and
// DeleteUser's cascade is one of them. This asserts that rather than inferring
// it: every user-keyed table must be empty afterwards.
func TestDeleteUserCascadeStaysAtomic(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close() //nolint:errcheck
	ctx := context.Background()

	user, err := p.AddUser(ctx, &schemas.User{
		Email:         refs.NewStringRef(uuid.New().String() + "@authorizer.dev"),
		SignupMethods: constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)

	orgID := uuid.New().String()
	require.NoError(t, p.AddSession(ctx, &schemas.Session{UserID: user.ID}))
	_, err = p.AddFederatedIdentity(ctx, &schemas.FederatedIdentity{
		OrgID: orgID, Issuer: "https://idp.example.com", Subject: uuid.New().String(), UserID: user.ID,
	})
	require.NoError(t, err)
	_, err = p.AddOrgMembership(ctx, &schemas.OrgMembership{OrgID: orgID, UserID: user.ID, Roles: "member"})
	require.NoError(t, err)
	_, err = p.AddAuthenticator(ctx, &schemas.Authenticator{
		UserID: user.ID, Method: constants.EnvKeyTOTPAuthenticator, Secret: "s3cret",
	})
	require.NoError(t, err)
	_, err = p.AddWebauthnCredential(ctx, &schemas.WebauthnCredential{
		UserID: user.ID, CredentialID: uuid.New().String(), PublicKey: "pk", Name: "laptop",
	})
	require.NoError(t, err)
	require.NoError(t, p.AddSessionToken(ctx, &schemas.SessionToken{UserID: user.ID, KeyName: "access", Token: "t"}))
	require.NoError(t, p.AddMFASession(ctx, &schemas.MFASession{UserID: user.ID, KeyName: "mfa"}))

	require.NoError(t, p.DeleteUser(ctx, user))

	for _, table := range schemas.UserOwnedCollections {
		var count int64
		require.NoError(t, p.DB().WithContext(ctx).Table(table).Where("user_id = ?", user.ID).Count(&count).Error)
		assert.Zero(t, count, "%s still holds rows for the deleted user", table)
	}
}

// TestCascadeDeleteSurvivesContention covers the explicit transactions. Their
// statements cannot be retried individually once SereneDB aborts the
// transaction with 40001, so the SQL provider replays the whole block
// (sql.Options.RetryTransactions). Concurrent cascades over the same rows must
// all succeed and leave nothing behind.
func TestCascadeDeleteSurvivesContention(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close() //nolint:errcheck
	ctx := context.Background()

	for round := 0; round < 5; round++ {
		user, err := p.AddUser(ctx, &schemas.User{
			Email:         refs.NewStringRef(uuid.New().String() + "@authorizer.dev"),
			SignupMethods: constants.AuthRecipeMethodBasicAuth,
		})
		require.NoError(t, err)
		for i := 0; i < 5; i++ {
			require.NoError(t, p.AddSession(ctx, &schemas.Session{UserID: user.ID}))
		}

		const workers = 8
		errs := make([]error, workers)
		var wg sync.WaitGroup
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				errs[i] = p.DeleteUser(ctx, user)
			}(i)
		}
		wg.Wait()
		for i, err := range errs {
			require.NoError(t, err, "round %d worker %d", round, i)
		}

		var count int64
		require.NoError(t, p.DB().WithContext(ctx).Table("authorizer_sessions").Where("user_id = ?", user.ID).Count(&count).Error)
		assert.Zero(t, count)
		_, err = p.GetUserByID(ctx, user.ID)
		assert.Error(t, err, "user must be gone")
	}
}

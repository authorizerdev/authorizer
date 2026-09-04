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

// newTestProvider connects to the SereneDB container started by
// `make test-serenedb`. Gated on TEST_DBS first, the way the SQL migration
// tests are, so `make test` (TEST_DBS=sqlite) stays Docker-free even on a
// machine that happens to have something bound to :7890.
func newTestProvider(t *testing.T) *provider {
	t.Helper()
	if !slices.Contains(strings.Split(os.Getenv("TEST_DBS"), ","), constants.DbTypeSereneDB) {
		t.Skip("set TEST_DBS=serenedb (make test-serenedb) to run SereneDB tests")
	}
	conn, err := net.DialTimeout("tcp", "localhost:7890", 2*time.Second)
	if err != nil {
		t.Skipf("skipping SereneDB tests: not reachable on localhost:7890: %v", err)
	}
	_ = conn.Close()

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	p, err := NewProvider(&config.Config{
		DatabaseType: constants.DbTypeSereneDB,
		DatabaseURL:  testDBURL,
		DatabaseName: "authorizer_test",
	}, &Dependencies{Log: &logger})
	require.NoError(t, err)
	require.NotNil(t, p)
	return p
}

// TestMigrateIsRepeatable is the restart path. GORM AutoMigrate fails here on
// the second boot — SereneDB reports varchar(n) as text, so AutoMigrate tries
// ALTER COLUMN ... TYPE and SereneDB refuses it on an indexed column. The
// create-only migrator must be a no-op once the schema exists.
func TestMigrateIsRepeatable(t *testing.T) {
	p := newTestProvider(t)
	require.NoError(t, p.Close())

	for i := 0; i < 2; i++ {
		p := newTestProvider(t)
		require.NoError(t, p.Close())
	}
}

// TestAddVerificationRequestUpsert covers the ON CONFLICT (email, identifier)
// fallback: SereneDB rejects a unique index as a conflict target, so the second
// request must be turned into an UPDATE off the 23505, not surface as an error
// or leave a duplicate row.
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
// fallback. The check-then-insert in AddAuthenticator has a race; on PostgreSQL
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

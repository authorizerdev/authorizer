package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestCreateSessionToken_PreservesAuthTimeAcrossRollover is a regression test
// for the oidcc-prompt-none-logged-in / oidcc-max-age-10000 conformance
// failures: a session rollover (silent SSO continuation) must NOT reset
// auth_time, even though IssuedAt legitimately refreshes every call.
func TestCreateSessionToken_PreservesAuthTimeAcrossRollover(t *testing.T) {
	p := &provider{config: &config.Config{ClientSecret: "test-secret"}}
	user := &schemas.User{ID: "user-1"}

	originalLogin := time.Now().Add(-5 * time.Minute).Unix()

	// First mint (real login): no AuthTime supplied, defaults to now.
	firstSession, _, _, err := p.CreateSessionToken(&AuthTokenConfig{User: user})
	require.NoError(t, err)
	require.NotZero(t, firstSession.AuthTime)

	// Rollover: caller explicitly threads the ORIGINAL auth time forward,
	// as authorize.go's rollover call sites now do via EffectiveAuthTime().
	rolledSession, _, _, err := p.CreateSessionToken(&AuthTokenConfig{User: user, AuthTime: originalLogin})
	require.NoError(t, err)

	assert.Equal(t, originalLogin, rolledSession.AuthTime, "AuthTime must survive rollover unchanged")
	assert.NotEqual(t, originalLogin, rolledSession.IssuedAt, "IssuedAt must still refresh on rollover")
}

func TestSessionData_EffectiveAuthTime_FallsBackToIssuedAt(t *testing.T) {
	// Pre-fix session cookie: AuthTime never existed, unmarshals to zero value.
	old := &SessionData{IssuedAt: 12345}
	assert.Equal(t, int64(12345), old.EffectiveAuthTime())

	fixed := &SessionData{IssuedAt: 12345, AuthTime: 999}
	assert.Equal(t, int64(999), fixed.EffectiveAuthTime())
}

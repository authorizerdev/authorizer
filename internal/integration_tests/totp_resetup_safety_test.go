package integration_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestTOTPResetupDoesNotDesyncUntilConfirmed guards the re-setup safety fix: a
// user re-running TOTP setup on an already-verified authenticator must keep
// their existing authenticator app working until they actually confirm the new
// code. The new secret is staged as pending and only promoted to the live row
// on confirmation, so an abandoned re-setup can never lock the account out.
func TestTOTPResetupDoesNotDesyncUntilConfirmed(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	ts := initTestSetup(t, cfg)
	require.NotNil(t, ts.AuthenticatorProvider, "TOTP must be enabled for this test")
	_, ctx := createContext(ts)

	email := "totp_resetup_" + uuid.NewString() + "@authorizer.dev"
	now := time.Now().Unix()
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:           refs.NewStringRef(email),
		EmailVerifiedAt: &now,
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)

	getRow := func() *schemas.Authenticator {
		row, gErr := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
		require.NoError(t, gErr)
		require.NotNil(t, row)
		return row
	}

	// Initial enrollment + confirmation → row is verified with secret #1.
	enroll1, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
	require.NoError(t, err)
	code1, err := totp.GenerateCode(enroll1.Secret, time.Now())
	require.NoError(t, err)
	ok, err := ts.AuthenticatorProvider.Validate(ctx, code1, user.ID)
	require.NoError(t, err)
	require.True(t, ok, "initial enrollment code must confirm")
	liveSecret := getRow().Secret
	require.NotNil(t, getRow().VerifiedAt)

	// Re-setup: generate secret #2. The live row must be UNTOUCHED — this is the
	// bug being fixed (previously the live secret was overwritten immediately).
	enroll2, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
	require.NoError(t, err)
	require.NotEqual(t, enroll1.Secret, enroll2.Secret, "re-setup must generate a fresh secret")
	assert.Equal(t, liveSecret, getRow().Secret, "re-setup must NOT overwrite the live secret before confirmation")

	// The previously-working authenticator must keep validating.
	oldCode, err := totp.GenerateCode(enroll1.Secret, time.Now())
	require.NoError(t, err)
	ok, err = ts.AuthenticatorProvider.Validate(ctx, oldCode, user.ID)
	require.NoError(t, err)
	assert.True(t, ok, "the old authenticator must keep working until the new code is confirmed")
	assert.Equal(t, liveSecret, getRow().Secret, "validating the old code must not promote the pending secret")

	// Confirm the new code → the pending secret is promoted to the live row.
	newCode, err := totp.GenerateCode(enroll2.Secret, time.Now())
	require.NoError(t, err)
	ok, err = ts.AuthenticatorProvider.Validate(ctx, newCode, user.ID)
	require.NoError(t, err)
	require.True(t, ok, "confirming the new code must succeed")
	assert.NotEqual(t, liveSecret, getRow().Secret, "after confirmation the new secret must be live")

	// The old secret must no longer validate; the new one must.
	oldCodeAfter, err := totp.GenerateCode(enroll1.Secret, time.Now())
	require.NoError(t, err)
	ok, _ = ts.AuthenticatorProvider.Validate(ctx, oldCodeAfter, user.ID)
	assert.False(t, ok, "after promotion the old secret must no longer validate")

	newCodeAfter, err := totp.GenerateCode(enroll2.Secret, time.Now())
	require.NoError(t, err)
	ok, err = ts.AuthenticatorProvider.Validate(ctx, newCodeAfter, user.ID)
	require.NoError(t, err)
	assert.True(t, ok, "the newly-confirmed secret must validate")
}

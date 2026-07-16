package integration_tests

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestEvaluateMFAGateForOAuth covers oauth_callback.go's entry point into the
// same gate Login/SignUp/WebauthnLoginVerify use, exercised directly at the
// service layer (no OAuth provider round-trip needed — the function only
// takes an already-resolved *schemas.User). See mfa_gate_test.go for
// resolveMFAGate's own decision table in isolation.
//
// NOTE: this does not exercise OAuthCallbackHandler itself (the HTTP
// handler) — there is no existing test double for a real OAuth
// provider round-trip (Google/GitHub/etc.) in this codebase, and building
// one is out of scope for this change. Full end-to-end coverage of the
// redirect built in oauth_callback.go requires manual/browser testing.
func TestEvaluateMFAGateForOAuth(t *testing.T) {
	newTestUser := func(t *testing.T, ts *testSetup, ctx context.Context, mutate func(*schemas.User)) *schemas.User {
		t.Helper()
		now := time.Now().Unix()
		user := &schemas.User{
			Email:           refs.NewStringRef("oauth_mfa_gate_" + uuid.NewString() + "@authorizer.dev"),
			EmailVerifiedAt: &now,
			SignupMethods:   constants.AuthRecipeMethodGoogle,
		}
		if mutate != nil {
			mutate(user)
		}
		created, err := ts.StorageProvider.AddUser(ctx, user)
		require.NoError(t, err)
		return created
	}

	t.Run("mfaGateNone does not withhold", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		user := newTestUser(t, ts, ctx, func(u *schemas.User) {
			u.IsMultiFactorAuthEnabled = refs.NewBoolRef(false)
		})

		side := &service.ResponseSideEffects{}
		meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts)}
		withheld, redirectSuffix, err := ts.ServiceProvider.EvaluateMFAGateForOAuth(ctx, meta, side, user)
		require.NoError(t, err)
		assert.False(t, withheld)
		assert.Empty(t, redirectSuffix)
		assert.Empty(t, side.Cookies, "no mfa session cookie should be set when the gate does not withhold")
	})

	t.Run("mfaGateSkippedSetup does not withhold", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		skippedAt := time.Now().Unix()
		user := newTestUser(t, ts, ctx, func(u *schemas.User) {
			u.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
			u.HasSkippedMFASetupAt = &skippedAt
		})

		side := &service.ResponseSideEffects{}
		meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts)}
		withheld, redirectSuffix, err := ts.ServiceProvider.EvaluateMFAGateForOAuth(ctx, meta, side, user)
		require.NoError(t, err)
		assert.False(t, withheld)
		assert.Empty(t, redirectSuffix)
	})

	t.Run("mfaGateOfferAll withholds with mfa_required=1", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		user := newTestUser(t, ts, ctx, func(u *schemas.User) {
			u.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		})

		side := &service.ResponseSideEffects{}
		meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts)}
		withheld, redirectSuffix, err := ts.ServiceProvider.EvaluateMFAGateForOAuth(ctx, meta, side, user)
		require.NoError(t, err)
		assert.True(t, withheld)
		assert.Contains(t, redirectSuffix, "mfa_required=1")
		assert.NotEmpty(t, side.Cookies, "an mfa session cookie must be set on a withheld outcome")
	})

	t.Run("mfaGateOfferAll omits totp when TOTP login is disabled", func(t *testing.T) {
		// Regression guard for the oauth gate appending totp unconditionally:
		// every other method is gated on its own config flag, but totp used to
		// be listed even on a server where TOTP login is off (DisableTOTPLogin).
		// Enable WebAuthn so the offer branch still produces a non-empty
		// mfa_methods, then assert it lists only webauthn — never totp.
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = false
		cfg.EnableWebauthnMFA = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		user := newTestUser(t, ts, ctx, func(u *schemas.User) {
			u.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		})

		side := &service.ResponseSideEffects{}
		meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts)}
		withheld, redirectSuffix, err := ts.ServiceProvider.EvaluateMFAGateForOAuth(ctx, meta, side, user)
		require.NoError(t, err)
		assert.True(t, withheld)

		values, parseErr := url.ParseQuery(redirectSuffix)
		require.NoError(t, parseErr)
		methods := values.Get("mfa_methods")
		assert.NotContains(t, methods, constants.EnvKeyTOTPAuthenticator, "totp must not be offered when TOTP login is disabled")
		assert.Contains(t, methods, constants.AuthRecipeMethodWebauthn, "the offer branch must still list configured methods")
	})

	t.Run("mfaGateBlockEnroll withholds with mfa_required=1", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnforceMFA = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		user := newTestUser(t, ts, ctx, func(u *schemas.User) {
			u.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		})

		side := &service.ResponseSideEffects{}
		meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts)}
		withheld, redirectSuffix, err := ts.ServiceProvider.EvaluateMFAGateForOAuth(ctx, meta, side, user)
		require.NoError(t, err)
		assert.True(t, withheld)
		assert.Contains(t, redirectSuffix, "mfa_required=1")
	})

	t.Run("mfaGateBlockVerify withholds with mfa_required=1", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		// Enable every other MFA method the offer/enroll branches would list
		// (webauthn, email OTP, SMS OTP — IsSMSServiceEnabled is already true
		// in getTestConfig) so that if the gate mis-routed this verified-TOTP
		// user to mfaGateOfferAll/BlockEnroll instead of mfaGateBlockVerify,
		// mfa_methods would pick up those extra, un-enrolled methods too —
		// making "totp" alone an insufficient signal. asserting the exact
		// mfa_methods value below is what actually distinguishes the verify
		// branch (lists only the user's already-verified factors) from the
		// offer/enroll branches (lists every configured method).
		cfg.EnableWebauthnMFA = true
		cfg.EnableEmailOTP = true
		cfg.IsEmailServiceEnabled = true
		cfg.EnableSMSOTP = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		user := newTestUser(t, ts, ctx, func(u *schemas.User) {
			u.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		})
		now := time.Now().Unix()
		_, err := ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
			UserID:     user.ID,
			Method:     constants.EnvKeyTOTPAuthenticator,
			Secret:     "dummy-secret-for-oauth-gate-test",
			VerifiedAt: &now,
		})
		require.NoError(t, err)

		side := &service.ResponseSideEffects{}
		meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts)}
		withheld, redirectSuffix, err := ts.ServiceProvider.EvaluateMFAGateForOAuth(ctx, meta, side, user)
		require.NoError(t, err)
		assert.True(t, withheld)
		assert.Contains(t, redirectSuffix, "mfa_required=1")

		values, parseErr := url.ParseQuery(redirectSuffix)
		require.NoError(t, parseErr)
		// Exactly "totp" -- not "totp,webauthn,email_otp,sms_otp", which is
		// what an offer/enroll branch would produce given the config above.
		// This is what distinguishes mfaGateBlockVerify (only the factors
		// this user actually has verified) from mfaGateOfferAll/BlockEnroll
		// (every method the server has configured).
		assert.Equal(t, constants.EnvKeyTOTPAuthenticator, values.Get("mfa_methods"))
	})

	t.Run("locked user is rejected", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		lockedAt := time.Now().Unix()
		user := newTestUser(t, ts, ctx, func(u *schemas.User) {
			u.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
			u.MFALockedAt = &lockedAt
		})

		side := &service.ResponseSideEffects{}
		meta := service.RequestMetadata{HostURL: testAuthorizerHost(ts)}
		withheld, redirectSuffix, err := ts.ServiceProvider.EvaluateMFAGateForOAuth(ctx, meta, side, user)
		require.Error(t, err)
		assert.False(t, withheld)
		assert.Empty(t, redirectSuffix)

		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr))
		assert.Equal(t, service.KindFailedPrecondition, svcErr.Kind)
	})
}

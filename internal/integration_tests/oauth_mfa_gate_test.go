package integration_tests

import (
	"context"
	"errors"
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
		assert.Contains(t, redirectSuffix, "totp")
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

package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSignup tests the signup functionality of the Authorizer application.
func TestSignup(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "signup_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	t.Run("should fail for missing email or phone number", func(t *testing.T) {
		signupReq := &model.SignUpRequest{
			Password: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail for missing confirm password", func(t *testing.T) {
		signupReq := &model.SignUpRequest{
			Email:    &email,
			Password: password,
		}

		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail for mismatch confirm password", func(t *testing.T) {
		signupReq := &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: "test@123",
		}

		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail for weak password", func(t *testing.T) {
		signupReq := &model.SignUpRequest{
			Email:           &email,
			Password:        "test",
			ConfirmPassword: "test",
		}

		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail for invalid email", func(t *testing.T) {
		invalidEmail := "test"
		signupReq := &model.SignUpRequest{
			Email:           &invalidEmail,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail for invalid mobile number", func(t *testing.T) {
		invalidMobileNumber := "1243234"
		signupReq := &model.SignUpRequest{
			PhoneNumber:     &invalidMobileNumber,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should pass for valid email", func(t *testing.T) {
		signupReq := &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res.User)

		t.Run("should fail for duplicate email", func(t *testing.T) {
			signupReq := &model.SignUpRequest{
				Email:           &email,
				Password:        password,
				ConfirmPassword: password,
			}
			res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
			assert.Error(t, err)
			assert.Nil(t, res)
		})
	})

	t.Run("should fail when signup is disabled", func(t *testing.T) {
		cfg2 := getTestConfig()
		cfg2.EnableSignup = false
		ts2 := initTestSetup(t, cfg2)
		_, ctx2 := createContext(ts2)

		disabledEmail := "signup_disabled_" + uuid.New().String() + "@authorizer.dev"
		signupReq := &model.SignUpRequest{
			Email:           &disabledEmail,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts2.GraphQLProvider.SignUp(ctx2, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should use default scopes when empty scope list is provided", func(t *testing.T) {
		emptyEmail := "signup_empty_scope_" + uuid.New().String() + "@authorizer.dev"
		signupReq := &model.SignUpRequest{
			Email:           &emptyEmail,
			Password:        password,
			ConfirmPassword: password,
			Scope:           []string{},
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		require.NotNil(t, res.AccessToken)
		assert.NotEmpty(t, *res.AccessToken)

		// Parse access token and verify it contains default scopes
		claims, err := ts.TokenProvider.ParseJWTToken(*res.AccessToken)
		assert.NoError(t, err)
		scopeRaw, ok := claims["scope"]
		assert.True(t, ok, "access token must contain scope claim")
		scopeSlice, ok := scopeRaw.([]interface{})
		assert.True(t, ok, "scope claim must be an array")
		scopes := make([]string, len(scopeSlice))
		for i, s := range scopeSlice {
			scopes[i] = s.(string)
		}
		assert.Contains(t, scopes, "openid")
		assert.Contains(t, scopes, "email")
		assert.Contains(t, scopes, "profile")
	})

	t.Run("should pass for valid mobile number", func(t *testing.T) {
		mobileNumber := fmt.Sprintf("%d", time.Now().Unix())
		signupReq := &model.SignUpRequest{
			PhoneNumber:     &mobileNumber,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		// Validate mobile number
		assert.Equal(t, mobileNumber, *res.User.PhoneNumber)
		assert.True(t, res.User.PhoneNumberVerified)
		// Auth formula should be basic auth based on mobile number
		assert.Contains(t, constants.AuthRecipeMethodMobileBasicAuth, res.User.SignupMethods)

		t.Run("should fail for duplicate mobile number", func(t *testing.T) {
			signupReq := &model.SignUpRequest{
				PhoneNumber:     &mobileNumber,
				Password:        password,
				ConfirmPassword: password,
			}
			res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
			assert.Error(t, err)
			assert.Nil(t, res)
		})
	})
}

// TestSignupDefaultsMultiFactorAuthEnabled guards the regression where a new
// user's IsMultiFactorAuthEnabled stayed false by default even when MFA is
// available server-wide (EnableMFA) and not explicitly disabled - meaning the
// optional-MFA-with-skip offer flow (resolveMFAGate) never had anything to
// offer for the common, non-enforced case.
func TestSignupDefaultsMultiFactorAuthEnabled(t *testing.T) {
	t.Run("MFA available, not enforced, no explicit param - defaults to enabled", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		cfg.EnforceMFA = false
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "signup_mfa_default_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"
		res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.NotNil(t, res)

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.True(t, refs.BoolValue(user.IsMultiFactorAuthEnabled), "a new user must default into MFA when it's available and not disabled, so the optional-setup-with-skip flow has something to offer")
	})

	t.Run("MFA not available server-wide - new user does not default to enabled", func(t *testing.T) {
		// signup.go reads the single already-derived EnableMFA flag, not
		// DisableMFA directly - in production, Finalize() forces EnableMFA
		// false whenever DisableMFA is set, before signup.go ever runs.
		cfg := getTestConfig()
		cfg.EnableMFA = false
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "signup_mfa_killswitch_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.False(t, refs.BoolValue(user.IsMultiFactorAuthEnabled))
	})

	t.Run("explicit opt-out is respected over the new default", func(t *testing.T) {
		cfg := getTestConfig()
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "signup_mfa_explicit_opt_out_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"
		explicit := false
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
			IsMultiFactorAuthEnabled: &explicit,
		})
		require.NoError(t, err)

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.False(t, refs.BoolValue(user.IsMultiFactorAuthEnabled), "explicit opt-out must still be respected")
	})

	t.Run("EnforceMFA still forces enabled regardless of an explicit opt-out", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnforceMFA = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "signup_mfa_enforced_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"
		explicit := false
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
			IsMultiFactorAuthEnabled: &explicit,
		})
		require.NoError(t, err)

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.True(t, refs.BoolValue(user.IsMultiFactorAuthEnabled), "EnforceMFA must override even an explicit opt-out")
	})
}

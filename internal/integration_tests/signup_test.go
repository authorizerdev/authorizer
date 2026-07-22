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

// TestSignUpGatesToken verifies that a brand-new signup, like login, has its
// token withheld and is offered the first-time MFA setup screen when MFA is
// available server-wide, and issued a token immediately when it is not.
func TestSignUpGatesToken(t *testing.T) {
	const password = "Password@123"

	t.Run("MFA available, no explicit opt-out -> token withheld, offer all", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "signup_gate_offer_" + uuid.New().String() + "@authorizer.dev"
		res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Nil(t, res.AccessToken, "a brand-new signup with MFA available must withhold the token, same as login")
		assert.True(t, refs.BoolValue(res.ShouldShowTotpScreen))
		assert.NotNil(t, res.AuthenticatorSecret)
	})

	t.Run("MFA not available -> token issued immediately", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = false
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "signup_gate_none_" + uuid.New().String() + "@authorizer.dev"
		res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotNil(t, res.AccessToken)
	})

	// Security regression: signup is unauthenticated (authorizer.v1.public =
	// true). A caller explicitly setting IsMultiFactorAuthEnabled: false on
	// their own signup request must NOT be able to opt their new account out
	// of the server's MFA-on-by-default policy — that field is honored only
	// on the authenticated admin _update_user path, never here.
	t.Run("MFA available, caller sets IsMultiFactorAuthEnabled=false -> gate still applies", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "signup_gate_bypass_attempt_" + uuid.New().String() + "@authorizer.dev"
		res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(false),
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Nil(t, res.AccessToken, "a client-supplied IsMultiFactorAuthEnabled=false must not bypass the server's MFA-on-by-default policy")
		assert.True(t, refs.BoolValue(res.ShouldShowTotpScreen))
	})

	// Regression guard for finding I1 (final whole-branch review): this
	// block used to be guarded by `p.Config.EnableMFA && p.Config.EnableTOTPLogin`,
	// mirroring login.go's old guard. A server configured for WebAuthn-only
	// enforced MFA (EnableTOTPLogin off, EnableWebauthnMFA on) skipped the
	// gate entirely and issued a token to a brand-new signup unconditionally
	// -- no offer, no enforcement. The gate must now run whenever MFA
	// applies at all, and only the TOTP-specific parts of the response
	// should be conditioned on EnableTOTPLogin.
	t.Run("WebAuthn-only enforced MFA, unenrolled -> token withheld via mfaGateBlockEnroll, WebAuthn setup offered", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = false
		cfg.EnableWebauthnMFA = true
		cfg.EnforceMFA = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "signup_gate_webauthn_only_" + uuid.New().String() + "@authorizer.dev"
		res, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Nil(t, res.AccessToken, "must not issue a token to a brand-new signup on a WebAuthn-only enforced-MFA server")
		assert.True(t, refs.BoolValue(res.ShouldOfferWebauthnMfaSetup))
		assert.False(t, refs.BoolValue(res.ShouldShowTotpScreen), "TOTP login is disabled server-wide; must not offer a screen the user can't complete")
		assert.Nil(t, res.AuthenticatorSecret, "must not generate a TOTP enrollment when TOTP login is disabled")
	})
}

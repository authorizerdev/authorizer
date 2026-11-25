package integration_tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/descope/virtualwebauthn"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// tokenRoles decodes a JWT payload and returns its roles claim.
func tokenRoles(t *testing.T, accessToken string) []string {
	t.Helper()
	parts := strings.Split(accessToken, ".")
	require.Len(t, parts, 3, "not a JWT")
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims struct {
		Roles []string `json:"roles"`
	}
	require.NoError(t, json.Unmarshal(payload, &claims))
	return claims.Roles
}

// TestMFARolesAreCarriedAcrossTheInterruption is the regression test for
// issue #301: login accepts `roles`, but when MFA interrupts the login the
// token is minted later by verify_otp / webauthn_login_verify /
// skip_mfa_setup, none of which receive the original request.
// issueAuthResponse used every role on the user, so an MFA login got a
// different (larger) role set than the identical non-MFA login.
//
// Like scope, the roles are CARRIED server-side, bound to the MFA session —
// never re-supplied by the (unauthenticated) completion endpoint.
func TestMFARolesAreCarriedAcrossTheInterruption(t *testing.T) {
	const password = "Password@123"
	userRoles := []string{"user", "editor", "viewer"}

	// setupTOTPUser signs up (MFA off, so a token is issued) a user holding
	// userRoles, enrolls a verified TOTP authenticator and opts the user into
	// MFA. Returns the email, the TOTP secret and the single-use recovery
	// codes, which verify_otp accepts in place of a passcode — used for repeat
	// logins so no test has to wait out a TOTP window.
	setupTOTPUser := func(t *testing.T, ts *testSetup) (string, string, []string) {
		t.Helper()
		_, ctx := createContext(ts)
		email := "mfa_roles_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password, Roles: userRoles,
		})
		require.NoError(t, err)
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)

		authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
		require.NoError(t, err)
		authenticator, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
		require.NoError(t, err)
		now := time.Now().Unix()
		authenticator.VerifiedAt = &now
		_, err = ts.StorageProvider.UpdateAuthenticator(ctx, authenticator)
		require.NoError(t, err)

		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)
		return email, authConfig.Secret, authConfig.RecoveryCodes
	}

	// loginThenVerifyTOTP runs login (expecting the TOTP challenge) and
	// completes it with verify_otp using otp, returning the token's roles.
	loginThenVerifyTOTP := func(t *testing.T, ts *testSetup, email, otp string, roles []string) []string {
		t.Helper()
		req, ctx := createContext(ts)
		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email: &email, Password: password, Roles: roles,
		})
		require.NoError(t, err)
		require.Nil(t, loginRes.AccessToken, "precondition: MFA must withhold the token")
		require.True(t, refs.BoolValue(loginRes.ShouldShowTotpScreen), "precondition: TOTP challenge")

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", latestMfaSessionCookie(ts)))
		verifyRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{
			Email: &email, Otp: otp, IsTotp: refs.NewBoolRef(true),
		})
		require.NoError(t, err)
		require.NotNil(t, verifyRes.AccessToken)
		return tokenRoles(t, *verifyRes.AccessToken)
	}

	newTS := func(t *testing.T) *testSetup {
		cfg := getTestConfig()
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		require.NotNil(t, ts.AuthenticatorProvider, "TOTP must be enabled for this test")
		return ts
	}

	t.Run("roles requested at login survive verify_otp", func(t *testing.T) {
		ts := newTS(t)
		email, secret, _ := setupTOTPUser(t, ts)
		ts.Config.EnableMFA = true

		code, err := totp.GenerateCode(secret, time.Now())
		require.NoError(t, err)
		got := loginThenVerifyTOTP(t, ts, email, code, []string{"editor"})
		assert.Equal(t, []string{"editor"}, got,
			"the roles requested (and validated) at login must be the roles of the token minted after MFA")
	})

	t.Run("no requested roles yields the default roles, same as a non-MFA login", func(t *testing.T) {
		ts := newTS(t)
		email, _, recoveryCodes := setupTOTPUser(t, ts)

		// Non-MFA baseline: MFA is still globally off and off for this user.
		_, ctx := createContext(ts)
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(false)
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)
		plainRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, plainRes.AccessToken)
		baseline := tokenRoles(t, *plainRes.AccessToken)

		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)
		ts.Config.EnableMFA = true

		got := loginThenVerifyTOTP(t, ts, email, recoveryCodes[0], nil)
		assert.Equal(t, baseline, got, "MFA and non-MFA logins must yield identical roles")
		assert.Equal(t, ts.Config.DefaultRoles, got)
	})

	t.Run("roles the user does not hold are still rejected at login", func(t *testing.T) {
		ts := newTS(t)
		email, _, _ := setupTOTPUser(t, ts)
		ts.Config.EnableMFA = true

		_, ctx := createContext(ts)
		res, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email: &email, Password: password, Roles: []string{"admin"},
		})
		require.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("carried roles are single-use and do not leak into a later session", func(t *testing.T) {
		ts := newTS(t)
		email, _, recoveryCodes := setupTOTPUser(t, ts)
		ts.Config.EnableMFA = true

		got := loginThenVerifyTOTP(t, ts, email, recoveryCodes[0], []string{"viewer"})
		assert.Equal(t, []string{"viewer"}, got)
		got = loginThenVerifyTOTP(t, ts, email, recoveryCodes[1], nil)
		assert.Equal(t, ts.Config.DefaultRoles, got)
	})

	t.Run("roles requested at login survive a passkey second factor", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableWebauthnMFA = true
		ts := initTestSetup(t, cfg)

		user, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		user.Roles = strings.Join(userRoles, ",")
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err := ts.StorageProvider.UpdateUser(t.Context(), user)
		require.NoError(t, err)
		cfg.EnableMFA = true

		email := refs.StringValue(user.Email)
		req, ctx := createContext(ts)
		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email: &email, Password: password, Roles: []string{"editor"},
		})
		require.NoError(t, err)
		require.Nil(t, loginRes.AccessToken)
		require.True(t, refs.BoolValue(loginRes.ShouldOfferWebauthnMfaVerify))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", latestMfaSessionCookie(ts)))

		optRes, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, &email)
		require.NoError(t, err)
		assertOpts, err := virtualwebauthn.ParseAssertionOptions(optRes.Options)
		require.NoError(t, err)
		assertResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertOpts)
		verifyRes, err := ts.GraphQLProvider.WebauthnLoginVerify(ctx, &model.WebauthnLoginVerifyRequest{Credential: assertResp})
		require.NoError(t, err)
		require.NotNil(t, verifyRes.AccessToken)
		assert.Equal(t, []string{"editor"}, tokenRoles(t, *verifyRes.AccessToken),
			"a passkey completing the login's MFA step must honour the roles requested at login")
	})
}

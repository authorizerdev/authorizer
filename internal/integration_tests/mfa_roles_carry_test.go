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
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
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

		// The passkey spent the session: it cannot be redeemed again.
		res, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{Email: &email})
		require.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("a passkey login ignores another user's verified mfa session", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableWebauthnMFA = true
		ts := initTestSetup(t, cfg)

		passkeyUser, rp, authenticator, credential := registerPasskeyForNewUser(t, ts)
		passkeyUser.Roles = strings.Join(userRoles, ",")
		_, err := ts.StorageProvider.UpdateUser(t.Context(), passkeyUser)
		require.NoError(t, err)

		// A different account's Verified session, carrying roles=[editor].
		_, ctx := createContext(ts)
		otherEmail := "mfa_roles_other_" + uuid.NewString() + "@authorizer.dev"
		_, err = ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &otherEmail, Password: password, ConfirmPassword: password, Roles: userRoles,
		})
		require.NoError(t, err)
		other, err := ts.StorageProvider.GetUserByEmail(ctx, otherEmail)
		require.NoError(t, err)
		otherSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(other.ID, otherSession,
			constants.MFASessionPurposeVerified, time.Now().Add(3*time.Minute).Unix()))
		require.NoError(t, ts.MemoryStoreProvider.SetCache("mfa_roles:"+otherSession, `["editor"]`, 180))

		req, ctx := createContext(ts)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", otherSession))
		optRes, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, nil)
		require.NoError(t, err)
		assertOpts, err := virtualwebauthn.ParseAssertionOptions(optRes.Options)
		require.NoError(t, err)
		assertResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertOpts)
		verifyRes, err := ts.GraphQLProvider.WebauthnLoginVerify(ctx, &model.WebauthnLoginVerifyRequest{Credential: assertResp})
		require.NoError(t, err)
		require.NotNil(t, verifyRes.AccessToken)
		assert.ElementsMatch(t, userRoles, tokenRoles(t, *verifyRes.AccessToken),
			"another account's carried roles must not apply to this passkey login")

		purpose, err := ts.MemoryStoreProvider.GetMfaSession(other.ID, otherSession)
		require.NoError(t, err, "the other account's session must be left untouched")
		assert.Equal(t, constants.MFASessionPurposeVerified, purpose)
		blob, err := ts.MemoryStoreProvider.GetCache("mfa_roles:" + otherSession)
		require.NoError(t, err)
		assert.Equal(t, `["editor"]`, blob)
	})

	t.Run("roles survive TOTP enrollment from the login-time setup offer", func(t *testing.T) {
		ts := newTS(t)
		_, ctx := createContext(ts)
		email := "mfa_roles_enroll_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password, Roles: userRoles,
		})
		require.NoError(t, err)
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)
		ts.Config.EnableMFA = true

		// No factor enrolled and MFA not enforced: login offers setup.
		req, ctx := createContext(ts)
		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email: &email, Password: password, Roles: []string{"editor"},
		})
		require.NoError(t, err)
		require.Nil(t, loginRes.AccessToken)
		loginSession := latestMfaSessionCookie(ts)
		require.NotEmpty(t, loginSession)

		// Enroll via the MFA cookie (no bearer token) ...
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", loginSession))
		setupRes, err := ts.GraphQLProvider.TOTPMFASetup(ctx, &model.OtpMfaSetupRequest{Email: &email})
		require.NoError(t, err)
		require.NotNil(t, setupRes.AuthenticatorSecret)

		// ... and adopt the NEW session cookie the setup call issued, exactly
		// as a browser would.
		newSession := latestMfaSessionCookie(ts)
		require.NotEmpty(t, newSession)
		require.NotEqual(t, loginSession, newSession, "setup re-arms a fresh session")
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", newSession))
		code, err := totp.GenerateCode(*setupRes.AuthenticatorSecret, time.Now())
		require.NoError(t, err)
		verifyRes, err := ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{
			Email: &email, Otp: code, IsTotp: refs.NewBoolRef(true),
		})
		require.NoError(t, err)
		require.NotNil(t, verifyRes.AccessToken)
		assert.Equal(t, []string{"editor"}, tokenRoles(t, *verifyRes.AccessToken),
			"enrolling a factor mid-login must not drop the roles requested at login")

		// The roles were copied, not moved: the original login cookie (which
		// clients that ignore the new cookie keep using) still carries them.
		blob, err := ts.MemoryStoreProvider.GetCache("mfa_roles:" + loginSession)
		require.NoError(t, err)
		assert.Equal(t, `["editor"]`, blob)
	})

	// Unverified-email login: the roles request is carried unvalidated and
	// resolved only once the emailed code is proven.
	t.Run("unverified-email OTP branch resolves the roles request after the code", func(t *testing.T) {
		const knownOTP = "123456"
		run := func(t *testing.T, roles []string) (*model.AuthResponse, error) {
			cfg := getTestConfig()
			cfg.IsEmailServiceEnabled = true
			ts := initTestSetup(t, cfg)
			_, ctx := createContext(ts)
			email := "mfa_roles_unverified_" + uuid.NewString() + "@authorizer.dev"
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
			require.NoError(t, err)
			h := string(hash)
			_, err = ts.StorageProvider.AddUser(ctx, &schemas.User{
				Email: refs.NewStringRef(email), Password: &h,
				SignupMethods: constants.AuthRecipeMethodBasicAuth,
				Roles:         strings.Join(userRoles, ","),
			})
			require.NoError(t, err)

			req, ctx := createContext(ts)
			loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
				Email: &email, Password: password, Roles: roles,
			})
			require.NoError(t, err, "the unverified-email branch must not report invalid roles before the code is proven")
			require.True(t, refs.BoolValue(loginRes.ShouldShowEmailOtpScreen))

			_, err = ts.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
				Email:     email,
				Otp:       crypto.HashOTP(knownOTP, cfg.JWTSecret),
				ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
			})
			require.NoError(t, err)
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", latestMfaSessionCookie(ts)))
			return ts.GraphQLProvider.VerifyOTP(ctx, &model.VerifyOTPRequest{Email: &email, Otp: knownOTP})
		}

		t.Run("valid request", func(t *testing.T) {
			res, err := run(t, []string{"viewer"})
			require.NoError(t, err)
			require.NotNil(t, res.AccessToken)
			assert.Equal(t, []string{"viewer"}, tokenRoles(t, *res.AccessToken))
		})
		t.Run("invalid request is rejected only after the code is proven", func(t *testing.T) {
			res, err := run(t, []string{"admin"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid roles")
			assert.Nil(t, res)
		})
	})
}

package integration_tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// tokenScope decodes a JWT payload and returns its scope claim.
func tokenScope(t *testing.T, accessToken string) []string {
	t.Helper()
	parts := strings.Split(accessToken, ".")
	require.Len(t, parts, 3, "not a JWT")
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims struct {
		Scope []string `json:"scope"`
	}
	require.NoError(t, json.Unmarshal(payload, &claims))
	return claims.Scope
}

// TestMFAScopeIsCarriedAcrossTheInterruption is the regression test for scope
// loss through the MFA offer.
//
// Login and signup accept a scope, but when MFA is offered the token is
// withheld and minted LATER by skip_mfa_setup / verify_otp / webauthn — none of
// which receive the original request. issueAuthResponse therefore hardcoded
// ["openid","email","profile"], silently dropping every other scope the caller
// asked for. Delegation flows lost exactly the scopes they exist to attenuate,
// and the authorization-code state tuple
// (code@@challenge@@nonce@@redirectURI@@resource) had no scope to restore
// either.
//
// The scope must be CARRIED, never re-supplied by the client: skip_mfa_setup is
// unauthenticated, so accepting a scope there would let a caller self-grant
// privileges never requested at login.
func TestMFAScopeIsCarriedAcrossTheInterruption(t *testing.T) {
	const password = "Password@123"

	// setupPendingMFAUser signs up a user whose MFA offer is pending, and
	// returns the email plus the mfa session cookie value.
	setupPendingMFAUser := func(t *testing.T, ts *testSetup, scope []string) (string, string) {
		t.Helper()
		req, ctx := createContext(ts)
		email := "mfa_scope_" + uuid.NewString() + "@authorizer.dev"

		signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password, Scope: scope,
		})
		require.NoError(t, err)
		require.Nil(t, signupRes.AccessToken,
			"precondition: MFA is on, so signup must withhold the token")

		mfaSession := latestMfaSessionCookie(ts)
		require.NotEmpty(t, mfaSession, "signup must set an mfa session cookie")
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))
		return email, mfaSession
	}

	t.Run("a custom scope requested at signup survives skip_mfa_setup", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)

		requested := []string{"openid", "email", "profile", "read:invoices"}
		email, _ := setupPendingMFAUser(t, ts, requested)
		_, ctx := createContext(ts)
		// createContext resets the request; re-attach the cookie.
		req := ts.GinContext.Request
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", latestMfaSessionCookie(ts)))

		skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{Email: &email})
		require.NoError(t, err)
		require.NotNil(t, skipRes.AccessToken)

		assert.Equal(t, requested, tokenScope(t, *skipRes.AccessToken),
			"the scope requested at signup must survive the MFA interruption; "+
				"dropping it silently downgrades the token and breaks delegation flows")
	})

	t.Run("no requested scope still yields the default set", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)

		email, _ := setupPendingMFAUser(t, ts, nil)
		_, ctx := createContext(ts)
		req := ts.GinContext.Request
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", latestMfaSessionCookie(ts)))

		skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{Email: &email})
		require.NoError(t, err)
		require.NotNil(t, skipRes.AccessToken)

		assert.Equal(t, []string{"openid", "email", "profile"}, tokenScope(t, *skipRes.AccessToken),
			"a caller that asked for nothing must still get the default set — the fix is additive")
	})

	t.Run("a custom scope requested at login survives skip_mfa_setup", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		// Sign up and clear the first MFA offer so the user exists and can log in.
		email := "mfa_scope_login_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		ts.GinContext.Request.Header.Set("Cookie",
			fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", latestMfaSessionCookie(ts)))
		_, err = ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{Email: &email})
		require.NoError(t, err)

		// Force the offer again on the next login so the scope has an
		// interruption to survive.
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		user.HasSkippedMFASetupAt = nil
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		requested := []string{"openid", "email", "profile", "offline_access", "read:reports"}
		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email: &email, Password: password, Scope: requested,
		})
		require.NoError(t, err)
		require.Nil(t, loginRes.AccessToken, "precondition: login must withhold the token")

		ts.GinContext.Request.Header.Set("Cookie",
			fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", latestMfaSessionCookie(ts)))
		skipRes, err := ts.GraphQLProvider.SkipMFASetup(ctx, &model.SkipMfaSetupRequest{Email: &email})
		require.NoError(t, err)
		require.NotNil(t, skipRes.AccessToken)

		assert.Equal(t, requested, tokenScope(t, *skipRes.AccessToken),
			"the scope requested at login must survive the MFA interruption")
	})
}

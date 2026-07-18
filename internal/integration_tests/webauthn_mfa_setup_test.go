package integration_tests

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/descope/virtualwebauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestWebauthnRegistrationMfaSessionSetup covers registering a passkey during
// a token-withheld MFA offer (mfaGateOfferAll) via the MFA session cookie
// instead of a bearer token: it must complete the gate and issue the
// previously-withheld token, exactly like totp_mfa_setup +
// verify_otp(is_totp: true) does for TOTP. It also guards the security
// boundary this path adds: a caller who only proved a password (Verified
// session) must never be able to mint a brand-new passkey to skip challenging
// an EXISTING verified second factor (mfaGateBlockVerify).
func TestWebauthnRegistrationMfaSessionSetup(t *testing.T) {
	const password = "Password@123"

	t.Run("registers via MFA session, issues the withheld token, and quiets a later login", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableWebauthnMFA = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)
		rp := testRelyingParty(t, ts)
		credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

		email := "webauthn_mfa_setup_" + uuid.NewString() + "@authorizer.dev"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.Nil(t, loginRes.AccessToken, "first login with optional MFA and no prior enrollment/skip must withhold the token")
		require.True(t, refs.BoolValue(loginRes.ShouldOfferWebauthnMfaSetup))

		// No Authorization header at any point in this test — proves the
		// registration ceremony below authenticates via the MFA session cookie
		// alone, not a bearer token.
		mfaSession := latestMfaSessionCookie(ts)
		require.NotEmpty(t, mfaSession, "login must have set an mfa session cookie on the response")
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		optRes, err := ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, &email, nil)
		require.NoError(t, err)
		attOpts, err := virtualwebauthn.ParseAttestationOptions(optRes.Options)
		require.NoError(t, err)
		authenticator := virtualwebauthn.NewAuthenticatorWithOptions(virtualwebauthn.AuthenticatorOptions{
			UserHandle: []byte(attOpts.UserID),
		})
		authenticator.AddCredential(credential)
		attResp := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attOpts)

		verifyRes, err := ts.GraphQLProvider.WebauthnRegistrationVerify(ctx, &model.WebauthnRegistrationVerifyRequest{
			Credential: attResp,
			Email:      &email,
		})
		require.NoError(t, err)
		require.NotNil(t, verifyRes)
		require.NotNil(t, verifyRes.AccessToken, "registering a passkey mid-offer must issue the token that was withheld at login")
		assert.NotEmpty(t, *verifyRes.AccessToken)

		updated, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		creds, err := ts.StorageProvider.ListWebauthnCredentialsByUserID(ctx, updated.ID)
		require.NoError(t, err)
		assert.Len(t, creds, 1, "the credential must be persisted, not just the token issued")

		// Unlike skip_mfa_setup (which only records a timestamp), the passkey
		// just registered is a real second factor: authenticatorVerified is now
		// true, so login.go correctly moves the user from mfaGateOfferAll to
		// mfaGateBlockVerify and challenges it on the next login rather than
		// issuing a token outright — proof the credential was actually
		// persisted, not just the withheld token released once.
		secondLogin, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{Email: &email, Password: password})
		require.NoError(t, err)
		require.Nil(t, secondLogin.AccessToken, "a real second factor was just enrolled; the next login must challenge it, not skip straight to a token")
		assert.True(t, refs.BoolValue(secondLogin.ShouldOfferWebauthnMfaVerify), "the newly-registered passkey must be offered as the verify method on the next login")
	})

	t.Run("rejects with FailedPrecondition when the user already has a verified authenticator", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableWebauthnMFA = true
		cfg.EnableTOTPLogin = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "webauthn_mfa_setup_blocked_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		// A verified TOTP authenticator puts the user in mfaGateBlockVerify —
		// their own opted-in second factor. A caller who only proved a password
		// (Verified session, first factor only) must not be able to mint a
		// brand-new passkey to bypass challenging it.
		_, err = ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
			UserID:     user.ID,
			Method:     constants.EnvKeyTOTPAuthenticator,
			Secret:     "test-secret",
			VerifiedAt: &now,
		})
		require.NoError(t, err)

		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeVerified, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		_, err = ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, &email, nil)
		require.Error(t, err)

		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
		assert.Equal(t, service.KindFailedPrecondition, svcErr.Kind, "a user with a verified second factor must not be able to enroll a new passkey to bypass it")

		creds, err := ts.StorageProvider.ListWebauthnCredentialsByUserID(ctx, user.ID)
		require.NoError(t, err)
		assert.Len(t, creds, 0, "a rejected attempt must not have registered anything")
	})

	t.Run("rejects with Unauthenticated when caller has no valid mfa session and no bearer token", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableWebauthnMFA = true
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		email := "webauthn_mfa_setup_nosession_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		_, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		_, err = ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, &email, nil)
		require.Error(t, err)

		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
		assert.Equal(t, service.KindUnauthenticated, svcErr.Kind)
	})

	t.Run("rejects a Challenge session (ResendOTP/ForgotPassword) with Unauthenticated", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.EnableMFA = true
		cfg.EnableWebauthnMFA = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		email := "webauthn_mfa_setup_challenge_" + uuid.NewString() + "@authorizer.dev"
		now := time.Now().Unix()
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:                    refs.NewStringRef(email),
			EmailVerifiedAt:          &now,
			SignupMethods:            constants.AuthRecipeMethodBasicAuth,
			IsMultiFactorAuthEnabled: refs.NewBoolRef(true),
		})
		require.NoError(t, err)

		// A Challenge session (no first factor proven) must never be tradeable
		// for a registered credential, same as it can never be traded for a
		// token via SkipMFASetup/VerifyOTP.
		mfaSession := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, constants.MFASessionPurposeChallenge, time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		_, err = ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, &email, nil)
		require.Error(t, err)

		var svcErr *service.Error
		require.True(t, errors.As(err, &svcErr), "expected a *service.Error, got %T: %v", err, err)
		assert.Equal(t, service.KindUnauthenticated, svcErr.Kind, "a Challenge session must be rejected like a missing session")
	})

	t.Run("settings-page bearer-token registration is unaffected: AuthResponse carries no access_token", func(t *testing.T) {
		cfg := getTestConfig()
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)
		rp := testRelyingParty(t, ts)
		credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

		email := "webauthn_settings_page_" + uuid.NewString() + "@authorizer.dev"
		signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email: &email, Password: password, ConfirmPassword: password,
		})
		require.NoError(t, err)
		require.NotNil(t, signupRes.AccessToken)
		req.Header.Set("Authorization", "Bearer "+*signupRes.AccessToken)

		optRes, err := ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, nil, nil)
		require.NoError(t, err)
		attOpts, err := virtualwebauthn.ParseAttestationOptions(optRes.Options)
		require.NoError(t, err)
		authenticator := virtualwebauthn.NewAuthenticatorWithOptions(virtualwebauthn.AuthenticatorOptions{
			UserHandle: []byte(attOpts.UserID),
		})
		authenticator.AddCredential(credential)
		attResp := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attOpts)

		verifyRes, err := ts.GraphQLProvider.WebauthnRegistrationVerify(ctx, &model.WebauthnRegistrationVerifyRequest{Credential: attResp})
		require.NoError(t, err)
		require.NotNil(t, verifyRes)
		assert.Nil(t, verifyRes.AccessToken, "an already-authenticated settings-page caller has a token already; this path must not mint a second one")
		assert.NotEmpty(t, verifyRes.Message)
	})
}

package integration_tests

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/descope/virtualwebauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// testRelyingParty builds a virtualwebauthn RelyingParty matching the RP the
// server derives from the test HTTP server's own host — the two must agree
// (RPID + origin) for a simulated ceremony to verify successfully.
func testRelyingParty(t *testing.T, ts *testSetup) virtualwebauthn.RelyingParty {
	t.Helper()
	u, err := url.Parse(testAuthorizerHost(ts))
	require.NoError(t, err)
	return virtualwebauthn.RelyingParty{
		ID:     u.Hostname(),
		Name:   "Authorizer",
		Origin: u.Scheme + "://" + u.Host,
	}
}

// TestWebauthnPasskeyRegistrationAndLogin covers the full passkey lifecycle on
// an existing account: register, list, log in both usernameless (discoverable)
// and scoped (MFA-alternative), and delete with ownership enforced.
func TestWebauthnPasskeyRegistrationAndLogin(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	rp := testRelyingParty(t, ts)
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	// The authenticator is constructed once we know the server-assigned
	// WebAuthn user handle (below) — a real platform authenticator persists
	// this handle at registration and replays it on every discoverable
	// (usernameless) assertion; a blank handle is what a browser would send
	// for a *non*-resident credential, which FinishLogin correctly rejects for
	// the discoverable flow.
	var authenticator virtualwebauthn.Authenticator

	email := "webauthn_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, signupRes.AccessToken)
	req.Header.Set("Authorization", "Bearer "+*signupRes.AccessToken)

	var credentialID string

	t.Run("register a passkey for the authenticated user", func(t *testing.T) {
		optRes, err := ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, nil, nil)
		require.NoError(t, err)
		attOpts, err := virtualwebauthn.ParseAttestationOptions(optRes.Options)
		require.NoError(t, err)
		authenticator = virtualwebauthn.NewAuthenticatorWithOptions(virtualwebauthn.AuthenticatorOptions{
			UserHandle: []byte(attOpts.UserID),
		})
		authenticator.AddCredential(credential)
		attResp := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attOpts)
		name := "Test MacBook"
		verifyRes, err := ts.GraphQLProvider.WebauthnRegistrationVerify(ctx, &model.WebauthnRegistrationVerifyRequest{
			Name:       &name,
			Credential: attResp,
		})
		require.NoError(t, err)
		require.NotNil(t, verifyRes)
	})

	t.Run("registered passkey appears in the caller's own list", func(t *testing.T) {
		creds, err := ts.GraphQLProvider.WebauthnCredentials(ctx)
		require.NoError(t, err)
		require.Len(t, creds, 1)
		assert.Equal(t, "Test MacBook", creds[0].Name)
		credentialID = creds[0].ID
	})

	t.Run("usernameless (discoverable) login resolves the registering user", func(t *testing.T) {
		optRes, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, nil)
		require.NoError(t, err)
		assertOpts, err := virtualwebauthn.ParseAssertionOptions(optRes.Options)
		require.NoError(t, err)
		assertResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertOpts)
		authRes, err := ts.GraphQLProvider.WebauthnLoginVerify(ctx, &model.WebauthnLoginVerifyRequest{Credential: assertResp})
		require.NoError(t, err)
		require.NotNil(t, authRes.AccessToken)
		require.NotNil(t, authRes.User)
		assert.Equal(t, email, refs.StringValue(authRes.User.Email))
	})

	t.Run("scoped (MFA-alternative) login with email succeeds", func(t *testing.T) {
		require.NotNil(t, signupRes.User, "test needs the signed-up user's id to arm a real MFA session")
		mfaSession := uuid.New().String()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(signupRes.User.ID, mfaSession,
			constants.MFASessionPurposeVerified,
			time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		optRes, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, &email)
		require.NoError(t, err)
		assertOpts, err := virtualwebauthn.ParseAssertionOptions(optRes.Options)
		require.NoError(t, err)
		assertResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertOpts)
		authRes, err := ts.GraphQLProvider.WebauthnLoginVerify(ctx, &model.WebauthnLoginVerifyRequest{Credential: assertResp})
		require.NoError(t, err)
		require.NotNil(t, authRes.AccessToken)
	})

	t.Run("a second user cannot delete the first user's credential", func(t *testing.T) {
		email2 := "webauthn2_" + uuid.New().String() + "@authorizer.dev"
		signupRes2, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email2,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+*signupRes2.AccessToken)

		_, err = ts.GraphQLProvider.WebauthnDeleteCredential(ctx, credentialID)
		assert.Error(t, err, "must not be able to delete another user's credential")

		// The owner can still delete it once we restore their auth context.
		req.Header.Set("Authorization", "Bearer "+*signupRes.AccessToken)
		delRes, err := ts.GraphQLProvider.WebauthnDeleteCredential(ctx, credentialID)
		require.NoError(t, err)
		assert.NotEmpty(t, delRes.Message)

		creds, err := ts.GraphQLProvider.WebauthnCredentials(ctx)
		require.NoError(t, err)
		assert.Len(t, creds, 0, "credential must be gone after deletion")
	})
}

// TestWebauthnLoginRequiresVerifiedEmail guards the locked design decision that
// a passkey may not issue tokens for an account whose email is unverified,
// even though the credential and signature are perfectly valid — stricter than
// password login, applied consistently at the passkey login gate.
func TestWebauthnLoginRequiresVerifiedEmail(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	rp := testRelyingParty(t, ts)
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

	email := "webauthn_unverified_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, signupRes.AccessToken)
	req.Header.Set("Authorization", "Bearer "+*signupRes.AccessToken)

	// Register a passkey while the account is (test-config default) verified,
	// then flip the account back to unverified directly in storage to isolate
	// the login-time gate from signup/verification-flow plumbing.
	optRes, err := ts.GraphQLProvider.WebauthnRegistrationOptions(ctx, nil, nil)
	require.NoError(t, err)
	attOpts, err := virtualwebauthn.ParseAttestationOptions(optRes.Options)
	require.NoError(t, err)
	authenticator := virtualwebauthn.NewAuthenticatorWithOptions(virtualwebauthn.AuthenticatorOptions{
		UserHandle: []byte(attOpts.UserID),
	})
	authenticator.AddCredential(credential)
	attResp := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attOpts)
	_, err = ts.GraphQLProvider.WebauthnRegistrationVerify(ctx, &model.WebauthnRegistrationVerifyRequest{Credential: attResp})
	require.NoError(t, err)

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	user.EmailVerifiedAt = nil
	_, err = ts.StorageProvider.UpdateUser(ctx, user)
	require.NoError(t, err)

	loginOptRes, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, nil)
	require.NoError(t, err)
	assertOpts, err := virtualwebauthn.ParseAssertionOptions(loginOptRes.Options)
	require.NoError(t, err)
	assertResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertOpts)
	authRes, err := ts.GraphQLProvider.WebauthnLoginVerify(ctx, &model.WebauthnLoginVerifyRequest{Credential: assertResp})
	assert.Error(t, err, "passkey login must be refused while the account's email is unverified")
	assert.Nil(t, authRes)
	if err != nil {
		assert.Contains(t, err.Error(), "verif", "error should be the distinct, actionable email-verification message, not a generic invalid-credential error")
	}
}

// TestWebauthnLoginOptionsScopedRequiresMfaSession guards against passkey/user
// enumeration via the scoped (email-provided) webauthn_login_options: without
// proof of password authentication (the same MFA session cookie verify_otp
// requires for its own MFA-alternative flow), a caller must not be able to
// distinguish "this account has a passkey" from "it doesn't" - the real
// PublicKeyCredentialRequestOptions returned on success (including that
// account's own credential IDs) is itself the leak.
func TestWebauthnLoginOptionsScopedRequiresMfaSession(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	rp := testRelyingParty(t, ts)
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

	emailWithPasskey := "webauthn_enum_haspasskey_" + uuid.New().String() + "@authorizer.dev"
	emailNoAccount := "webauthn_enum_noaccount_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &emailWithPasskey,
		Password:        password,
		ConfirmPassword: password,
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
	_, err = ts.GraphQLProvider.WebauthnRegistrationVerify(ctx, &model.WebauthnRegistrationVerifyRequest{Credential: attResp})
	require.NoError(t, err)

	// No MFA session armed at all from here on - simulating an
	// unauthenticated caller who never completed password login.
	req.Header.Del("Authorization")
	req.Header.Del("Cookie")

	t.Run("account with a passkey - refused the same way as one without", func(t *testing.T) {
		_, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, &emailWithPasskey)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid session")
	})

	t.Run("account that doesn't exist - refused identically, not a different error", func(t *testing.T) {
		_, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, &emailNoAccount)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid session",
			"a nonexistent account must fail at the same session gate as a real one - not a distinguishable error")
	})

	t.Run("a stale MFA session for a DIFFERENT user cannot be reused to probe this account", func(t *testing.T) {
		otherUserID := uuid.New().String()
		mfaSession := uuid.New().String()
		require.NoError(t, ts.MemoryStoreProvider.SetMfaSession(otherUserID, mfaSession,
			constants.MFASessionPurposeVerified,
			time.Now().Add(5*time.Minute).Unix()))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.MfaCookieName+"_session", mfaSession))

		_, err := ts.GraphQLProvider.WebauthnLoginOptions(ctx, &emailWithPasskey)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid session",
			"a valid session for one user must not unlock scoped options for a different user")
	})
}

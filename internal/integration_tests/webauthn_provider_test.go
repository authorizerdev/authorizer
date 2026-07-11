package integration_tests

import (
	"context"
	"testing"
	"time"

	"github.com/descope/virtualwebauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// These exercise internal/authenticators/webauthn.Provider directly (via
// ts.WebAuthnProvider, the real instance the server uses) rather than through
// GraphQL, covering exactly what a second-pass review flagged as untested:
// sign-count regression rejection, a forged assertion signed with the wrong
// key, and challenge replay. They reuse the existing integration harness
// (real storage + memory store) instead of duplicating provider construction
// in a from-scratch package-level test file.

func mkWebauthnTestUser(t *testing.T, ts *testSetup, ctx context.Context) *schemas.User {
	t.Helper()
	email := "webauthn_provider_" + uuid.NewString() + "@authorizer.dev"
	now := time.Now().Unix()
	user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
		Email:           refs.NewStringRef(email),
		EmailVerifiedAt: &now,
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)
	return user
}

// TestWebauthnProviderSignCountRegressionRejected guards the fix for a
// documented-but-previously-nonfunctional control: go-webauthn does NOT error
// on a sign-count regression, it only sets credential.Authenticator.
// CloneWarning and still reports success - FinishLogin must check that flag
// itself and refuse the login, or cloned-authenticator detection (the entire
// reason sign_count is tracked) never actually rejects anything.
func TestWebauthnProviderSignCountRegressionRejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)
	host := testAuthorizerHost(ts)
	rp := testRelyingParty(t, ts)

	user := mkWebauthnTestUser(t, ts, ctx)

	regOptions, err := ts.WebAuthnProvider.BeginRegistration(ctx, host, user)
	require.NoError(t, err)
	attOpts, err := virtualwebauthn.ParseAttestationOptions(regOptions)
	require.NoError(t, err)

	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	credential.Counter = 1
	authenticator := virtualwebauthn.NewAuthenticatorWithOptions(virtualwebauthn.AuthenticatorOptions{
		UserHandle: []byte(attOpts.UserID),
	})
	authenticator.AddCredential(credential)
	attResp := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attOpts)
	_, err = ts.WebAuthnProvider.FinishRegistration(ctx, host, user, "test", attResp)
	require.NoError(t, err)

	// First login at counter=5 must succeed and persist sign_count=5.
	credential.Counter = 5
	loginOptions, err := ts.WebAuthnProvider.BeginLogin(ctx, host, user)
	require.NoError(t, err)
	assertOpts, err := virtualwebauthn.ParseAssertionOptions(loginOptions)
	require.NoError(t, err)
	assertResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertOpts)
	_, _, err = ts.WebAuthnProvider.FinishLogin(ctx, host, assertResp)
	require.NoError(t, err, "a genuine higher counter must be accepted")

	// Second login at counter=3 - LOWER than the last stored value (5) -
	// simulates a cloned authenticator replaying stale state. Without the
	// CloneWarning check this would silently succeed (go-webauthn does not
	// error here on its own).
	credential.Counter = 3
	loginOptions2, err := ts.WebAuthnProvider.BeginLogin(ctx, host, user)
	require.NoError(t, err)
	assertOpts2, err := virtualwebauthn.ParseAssertionOptions(loginOptions2)
	require.NoError(t, err)
	assertResp2 := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertOpts2)
	_, _, err = ts.WebAuthnProvider.FinishLogin(ctx, host, assertResp2)
	assert.Error(t, err, "a sign-count regression must be rejected - this is exactly what clone detection exists for")
}

// TestWebauthnProviderForgedAssertionRejected guards the discoverable-login
// resolution against spoofing: an assertion carrying the REAL registered
// credential ID but signed with a DIFFERENT (attacker's own) private key must
// be rejected. Resolution by credential_id alone would be a critical
// auth-bypass if the RP didn't then verify the signature against the
// genuinely stored public key.
func TestWebauthnProviderForgedAssertionRejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)
	host := testAuthorizerHost(ts)
	rp := testRelyingParty(t, ts)

	user := mkWebauthnTestUser(t, ts, ctx)

	regOptions, err := ts.WebAuthnProvider.BeginRegistration(ctx, host, user)
	require.NoError(t, err)
	attOpts, err := virtualwebauthn.ParseAttestationOptions(regOptions)
	require.NoError(t, err)

	realCredential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	authenticator := virtualwebauthn.NewAuthenticatorWithOptions(virtualwebauthn.AuthenticatorOptions{
		UserHandle: []byte(attOpts.UserID),
	})
	authenticator.AddCredential(realCredential)
	attResp := virtualwebauthn.CreateAttestationResponse(rp, authenticator, realCredential, *attOpts)
	_, err = ts.WebAuthnProvider.FinishRegistration(ctx, host, user, "test", attResp)
	require.NoError(t, err)

	// An attacker's credential: a fresh, unrelated keypair, but with its ID
	// overwritten to claim the victim's real, registered credential ID.
	forgedCredential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	forgedCredential.ID = realCredential.ID

	loginOptions, err := ts.WebAuthnProvider.BeginLogin(ctx, host, user)
	require.NoError(t, err)
	assertOpts, err := virtualwebauthn.ParseAssertionOptions(loginOptions)
	require.NoError(t, err)
	forgedResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, forgedCredential, *assertOpts)
	_, _, err = ts.WebAuthnProvider.FinishLogin(ctx, host, forgedResp)
	assert.Error(t, err, "an assertion claiming the real credential id but signed with a different key must be rejected")
}

// TestWebauthnProviderChallengeReplayRejected guards single-use enforcement
// on the login challenge: replaying the exact same (previously successful)
// assertion response a second time must be rejected, not silently accepted as
// a second valid login.
func TestWebauthnProviderChallengeReplayRejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)
	host := testAuthorizerHost(ts)
	rp := testRelyingParty(t, ts)

	user := mkWebauthnTestUser(t, ts, ctx)

	regOptions, err := ts.WebAuthnProvider.BeginRegistration(ctx, host, user)
	require.NoError(t, err)
	attOpts, err := virtualwebauthn.ParseAttestationOptions(regOptions)
	require.NoError(t, err)

	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	authenticator := virtualwebauthn.NewAuthenticatorWithOptions(virtualwebauthn.AuthenticatorOptions{
		UserHandle: []byte(attOpts.UserID),
	})
	authenticator.AddCredential(credential)
	attResp := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attOpts)
	_, err = ts.WebAuthnProvider.FinishRegistration(ctx, host, user, "test", attResp)
	require.NoError(t, err)

	loginOptions, err := ts.WebAuthnProvider.BeginLogin(ctx, host, user)
	require.NoError(t, err)
	assertOpts, err := virtualwebauthn.ParseAssertionOptions(loginOptions)
	require.NoError(t, err)
	assertResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertOpts)

	_, _, err = ts.WebAuthnProvider.FinishLogin(ctx, host, assertResp)
	require.NoError(t, err, "the first use of the assertion must succeed")

	_, _, err = ts.WebAuthnProvider.FinishLogin(ctx, host, assertResp)
	assert.Error(t, err, "replaying the same assertion/challenge a second time must be rejected")
}

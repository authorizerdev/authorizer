package sdktests

// WebAuthn / passkey.
//
// Unlike the browser suite (which needs Chrome's CDP virtual authenticator),
// the WebAuthn *authenticator* here is a pure-Go software authenticator
// (github.com/descope/virtualwebauthn — the standard pairing for go-webauthn,
// which is the server-side library Authorizer uses). That means the full
// registration ceremony can be completed WITHOUT a browser and driven through
// the SDK's typed methods:
//
//   WebauthnRegistrationOptions (SDK, cookie threaded) → virtualwebauthn signs
//   the attestation → WebauthnRegistrationVerify (SDK, cookie threaded).
//
// Both option/verify methods accept per-call headers, so the mfa_session cookie
// (which the withheld-token first-time-offer path requires) is threaded in and
// these are genuine SDK exercises.
//
// The passwordless passkey LOGIN completion is NOT SDK-drivable: the challenge
// from webauthn_login_options lives in a server-set session cookie that
// webauthn_login_options returns via Set-Cookie (discarded by the SDK) and
// webauthn_login_verify has no header parameter to send it back on — the same
// cookie gap that blocks verify_otp. Login is therefore covered only up to the
// options-shape assertion here; its full completion stays in the browser suite.
//
// The whole file requires the authorizer-webauthn instance reached over its
// pinned origin (http://webauthn.e2e-playground.test:8080) — go-webauthn's RPID
// validation and the instance's allowed-origins reject any other host. When run
// from the host (localhost) rather than inside the compose network, it skips.

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	authorizer "github.com/authorizerdev/authorizer-go/v2"
	"github.com/descope/virtualwebauthn"
)

// relyingParty derives the RP the software authenticator must present, from the
// webauthn instance URL. It skips when the target is localhost, because the
// server pins its origin/RPID to the dotted compose hostname regardless of how
// it is reached, so only the in-network run can satisfy the CSRF + RPID checks.
func relyingParty(t *testing.T) virtualwebauthn.RelyingParty {
	t.Helper()
	u, err := url.Parse(webauthnURL)
	if err != nil {
		t.Fatalf("parse webauthnURL: %v", err)
	}
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" {
		t.Skipf("WebAuthn requires the pinned compose origin (%s); run inside the docker network via the go-sdk-tests service", webauthnURL)
	}
	return virtualwebauthn.RelyingParty{
		Name:   "Authorizer",
		ID:     host,                      // e.g. webauthn.e2e-playground.test
		Origin: u.Scheme + "://" + u.Host, // e.g. http://webauthn.e2e-playground.test:8080
	}
}

func TestWebAuthn_FullRegistrationCeremonyThroughSDK(t *testing.T) {
	rp := relyingParty(t)
	c := userClient(t, webauthnURL)
	email := randomEmail("webauthn-reg")

	if _, err := c.SignUp(&authorizer.SignUpRequest{
		Email: &email, Password: testPassword, ConfirmPassword: testPassword,
	}); err != nil {
		t.Fatalf("SignUp: %v", err)
	}

	// Login to arm the mfa_session cookie (captured by the jar; SDK Login would
	// discard it), then thread it into the SDK registration methods.
	jar := jarClient(t)
	loginCapture(t, jar, webauthnURL, email)
	cookieHdr := mfaSessionCookieHeader(t, jar, webauthnURL)
	hdrs := map[string]string{"Cookie": cookieHdr}

	// 1. registration options (SDK)
	optsRes, err := c.WebauthnRegistrationOptions(&authorizer.WebauthnRegistrationOptionsRequest{Email: &email}, hdrs)
	if err != nil {
		t.Fatalf("WebauthnRegistrationOptions (SDK): %v", err)
	}
	attestationOpts, err := virtualwebauthn.ParseAttestationOptions(optsRes.Options)
	if err != nil {
		t.Fatalf("parse attestation options %q: %v", optsRes.Options, err)
	}

	// 2. software authenticator signs the attestation
	authenticator := virtualwebauthn.NewAuthenticator()
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	attestationResponse := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attestationOpts)

	// 3. registration verify (SDK)
	name := "go-virtual-authenticator"
	verifyRes, err := c.WebauthnRegistrationVerify(&authorizer.WebauthnRegistrationVerifyRequest{
		Name:       &name,
		Credential: attestationResponse,
		Email:      &email,
	}, hdrs)
	if err != nil {
		t.Fatalf("WebauthnRegistrationVerify (SDK): %v", err)
	}
	// On the mfa-session path the withheld token is issued once enrollment
	// completes; at minimum a non-error response with a message is returned.
	if verifyRes == nil {
		t.Fatalf("WebauthnRegistrationVerify returned nil response")
	}

	// 4. the passkey now appears in the caller's credential list (SDK query,
	//    authenticated with the token just issued, or the session cookie).
	authHdrs := hdrs
	if verifyRes.AccessToken != nil && *verifyRes.AccessToken != "" {
		authHdrs = map[string]string{"Authorization": "Bearer " + *verifyRes.AccessToken}
	}
	creds, err := c.WebauthnCredentials(authHdrs)
	if err != nil {
		t.Fatalf("WebauthnCredentials (SDK): %v", err)
	}
	if len(creds) == 0 {
		t.Fatalf("expected at least one registered passkey after ceremony")
	}
	found := false
	for _, cr := range creds {
		if cr.Name == name {
			found = true
		}
	}
	if !found {
		t.Errorf("registered passkey %q not found in credential list", name)
	}
}

// WebauthnLoginOptions is unauthenticated (start of passwordless login) and IS
// SDK-drivable; assert the challenge JSON is well-formed. Completing the login
// (webauthn_login_verify) is blocked by the cookie gap documented above.
func TestWebAuthn_LoginOptionsShapeThroughSDK(t *testing.T) {
	relyingParty(t) // skip gate for non-compose runs
	c := userClient(t, webauthnURL)

	res, err := c.WebauthnLoginOptions(nil) // discoverable / usernameless
	if err != nil {
		t.Fatalf("WebauthnLoginOptions (SDK): %v", err)
	}
	if strings.TrimSpace(res.Options) == "" {
		t.Fatalf("WebauthnLoginOptions returned empty options")
	}
	// Must parse as a valid assertion challenge.
	if _, err := virtualwebauthn.ParseAssertionOptions(res.Options); err != nil {
		// Some servers wrap as {publicKey:{...}}; ensure it is at least valid JSON
		// carrying a challenge field before failing hard.
		var probe map[string]json.RawMessage
		if json.Unmarshal([]byte(res.Options), &probe) != nil {
			t.Fatalf("login options not valid JSON: %q", res.Options)
		}
		t.Fatalf("ParseAssertionOptions: %v (options: %q)", err, res.Options)
	}
}

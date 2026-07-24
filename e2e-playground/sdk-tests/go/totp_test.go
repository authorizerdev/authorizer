package sdktests

// TOTP enrollment + challenge. Hybrid by necessity:
//
//   - SDK drives SignUp and TotpMfaSetup (the latter accepts per-call headers,
//     so the mfa_session cookie is threaded in and this is a genuine SDK
//     exercise — it builds the totp_mfa_setup mutation and parses
//     authenticator_secret, catching drift in that response shape).
//   - The raw jar client drives login (to capture the mfa_session Set-Cookie
//     the SDK's Login discards) and verify_otp (the SDK's VerifyOTP exposes no
//     header parameter, so the cookie verify_otp requires unconditionally can't
//     be sent through it). Both are labelled SDK gaps — see README.

import (
	"testing"

	"github.com/authorizerdev/authorizer-go/v2"
)

const verifyOTPMutation = `mutation ($params: VerifyOTPRequest!) {
	verify_otp(params: $params) { message access_token }
}`

// enrollTotp signs up, logs in (jar captures mfa_session), and runs
// TotpMfaSetup through the SDK, returning the jar client and the TOTP secret.
func enrollTotp(t *testing.T, email string) (*httpJar, string) {
	t.Helper()
	c := userClient(t, baseURL)
	if _, err := c.SignUp(&authorizer.SignUpRequest{
		Email: &email, Password: testPassword, ConfirmPassword: testPassword,
	}); err != nil {
		t.Fatalf("SignUp: %v", err)
	}

	jar := jarClient(t)
	login := loginCapture(t, jar, baseURL, email)
	if !boolValue(login.ShouldShowTotpScreen) {
		t.Fatalf("expected should_show_totp_screen=true, got %+v", login)
	}

	cookieHdr := mfaSessionCookieHeader(t, jar, baseURL)
	setup, err := c.TotpMfaSetup(&authorizer.TotpMfaSetupRequest{Email: &email},
		map[string]string{"Cookie": cookieHdr})
	if err != nil {
		t.Fatalf("TotpMfaSetup (SDK): %v", err)
	}
	if setup.AuthenticatorSecret == nil || *setup.AuthenticatorSecret == "" {
		t.Fatalf("TotpMfaSetup returned empty authenticator_secret")
	}
	return &httpJar{t, jar}, *setup.AuthenticatorSecret
}

func TestTOTP_EnrollAndVerify(t *testing.T) {
	email := randomEmail("totp")
	jar, secret := enrollTotp(t, email)

	res := jar.verifyOTP(map[string]any{"email": email, "otp": totpCode(t, secret), "is_totp": true})
	if len(res.Errors) > 0 {
		t.Fatalf("verify_otp with correct code errored: %s", res.errorText())
	}
	var wrap struct {
		VerifyOTP struct {
			AccessToken *string `json:"access_token"`
		} `json:"verify_otp"`
	}
	mustDecode(t, res, &wrap)
	if wrap.VerifyOTP.AccessToken == nil || *wrap.VerifyOTP.AccessToken == "" {
		t.Fatalf("expected access_token after successful TOTP verification")
	}
}

func TestTOTP_InvalidCodeRejected(t *testing.T) {
	email := randomEmail("totp-invalid")
	jar, secret := enrollTotp(t, email)

	res := jar.verifyOTP(map[string]any{"email": email, "otp": wrongTotpCode(t, secret), "is_totp": true})
	assertErrorContains(t, res, "invalid otp")
}

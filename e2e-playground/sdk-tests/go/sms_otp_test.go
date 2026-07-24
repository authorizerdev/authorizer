package sdktests

// SMS-OTP enrollment + challenge. Same hybrid split as TOTP: SDK drives SignUp
// and SmsOtpMfaSetup (accepts headers → genuine SDK call, threads the cookie and
// triggers the real code send to sms-sink); the raw jar client drives login
// (cookie capture) and verify_otp (no SDK header param). The code is read back
// from the sms-sink mock exactly as the Playwright suite does.

import (
	"testing"

	"github.com/authorizerdev/authorizer-go/v2"
)

func TestSMSOTP_EnrollAndVerify(t *testing.T) {
	c := userClient(t, baseURL)
	email := randomEmail("sms-otp")
	phone := randomPhone()

	if _, err := c.SignUp(&authorizer.SignUpRequest{
		Email: &email, PhoneNumber: &phone, Password: testPassword, ConfirmPassword: testPassword,
	}); err != nil {
		t.Fatalf("SignUp: %v", err)
	}

	jar := jarClient(t)
	login := loginCapture(t, jar, baseURL, email)
	if !boolValue(login.ShouldOfferSmsOtpMfaSetup) {
		t.Fatalf("expected should_offer_sms_otp_mfa_setup=true, got %+v", login)
	}

	cookieHdr := mfaSessionCookieHeader(t, jar, baseURL)
	if _, err := c.SmsOtpMfaSetup(&authorizer.SmsOtpMfaSetupRequest{PhoneNumber: &phone},
		map[string]string{"Cookie": cookieHdr}); err != nil {
		t.Fatalf("SmsOtpMfaSetup (SDK): %v", err)
	}

	code := extractOTP(t, pollSMS(t, phone))

	res := rawGraphQL(t, jar, baseURL, verifyOTPMutation, map[string]any{
		"params": map[string]any{"phone_number": phone, "otp": code, "is_totp": false},
	})
	var wrap struct {
		VerifyOTP struct {
			AccessToken *string `json:"access_token"`
		} `json:"verify_otp"`
	}
	mustDecode(t, res, &wrap)
	if wrap.VerifyOTP.AccessToken == nil || *wrap.VerifyOTP.AccessToken == "" {
		t.Fatalf("expected access_token after successful SMS-OTP verification")
	}
}

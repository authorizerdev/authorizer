package sdktests

// OTP brute-force lockout (#698): 5 failed verify_otp attempts within the
// sliding window lock the user out with a DISTINCT "too many failed attempts"
// error — increment-then-check, so the 6th call is the one rejected as locked,
// even with a correct code. Mirrors e2e-playground/tests/otp-lockout.spec.ts.
//
// Enrollment goes through the SDK (TotpMfaSetup / SmsOtpMfaSetup); the verify_otp
// attempts go through the raw jar client for the same SDK-gap reason as the
// TOTP/SMS happy-path tests (VerifyOTP takes no cookie header).

import (
	"testing"

	"github.com/authorizerdev/authorizer-go/v2"
)

func TestOTPLockout_TOTP(t *testing.T) {
	email := randomEmail("totp-lockout")
	jar, secret := enrollTotp(t, email)
	wrong := wrongTotpCode(t, secret)

	// 5 wrong codes → still the plain invalid-otp error.
	for i := 0; i < 5; i++ {
		res := jar.verifyOTP(map[string]any{"email": email, "otp": wrong, "is_totp": true})
		assertErrorContains(t, res, "invalid otp")
	}

	// 6th attempt (even wrong) → distinct lockout error, not invalid-otp.
	locked := jar.verifyOTP(map[string]any{"email": email, "otp": wrong, "is_totp": true})
	assertErrorContains(t, locked, "too many failed attempts")

	// The CORRECT code is also refused while locked — lockout blocks
	// verification outright, not just wrong guesses.
	stillLocked := jar.verifyOTP(map[string]any{"email": email, "otp": totpCode(t, secret), "is_totp": true})
	assertErrorContains(t, stillLocked, "too many failed attempts")
}

func TestOTPLockout_TOTPResetsAfterSuccess(t *testing.T) {
	email := randomEmail("totp-reset")
	jar, secret := enrollTotp(t, email)
	wrong := wrongTotpCode(t, secret)

	// 3 failed attempts (under the 5 budget), then succeed → counter cleared.
	for i := 0; i < 3; i++ {
		jar.verifyOTP(map[string]any{"email": email, "otp": wrong, "is_totp": true})
	}
	ok := jar.verifyOTP(map[string]any{"email": email, "otp": totpCode(t, secret), "is_totp": true})
	if len(ok.Errors) > 0 {
		t.Fatalf("expected success after 3 fails, got: %s", ok.errorText())
	}

	// Re-login (fresh mfa_session; lock key is per-user.ID so a stale count
	// would carry over) and burn a full 5 wrong attempts — if the reset hadn't
	// happened, 3 carried + 3 new would have locked before the 5th.
	jar2 := jarClient(t)
	loginCapture(t, jar2, baseURL, email)
	wrapped := &httpJar{t, jar2}
	for i := 0; i < 5; i++ {
		res := wrapped.verifyOTP(map[string]any{"email": email, "otp": wrong, "is_totp": true})
		assertErrorContains(t, res, "invalid otp")
	}
}

func TestOTPLockout_SMS(t *testing.T) {
	c := userClient(t, baseURL)
	email := randomEmail("sms-otp-lockout")
	phone := randomPhone()

	if _, err := c.SignUp(&authorizer.SignUpRequest{
		Email: &email, PhoneNumber: &phone, Password: testPassword, ConfirmPassword: testPassword,
	}); err != nil {
		t.Fatalf("SignUp: %v", err)
	}
	jar := jarClient(t)
	login := loginCapture(t, jar, baseURL, email)
	if !boolValue(login.ShouldOfferSmsOtpMfaSetup) {
		t.Fatalf("expected should_offer_sms_otp_mfa_setup=true")
	}
	cookieHdr := mfaSessionCookieHeader(t, jar, baseURL)
	if _, err := c.SmsOtpMfaSetup(&authorizer.SmsOtpMfaSetupRequest{PhoneNumber: &phone},
		map[string]string{"Cookie": cookieHdr}); err != nil {
		t.Fatalf("SmsOtpMfaSetup: %v", err)
	}
	correct := extractOTP(t, pollSMS(t, phone))
	const wrong = "ZZZZZZ" // outside the OTP charset window → always a mismatch

	for i := 0; i < 5; i++ {
		res := rawGraphQL(t, jar, baseURL, verifyOTPMutation, map[string]any{
			"params": map[string]any{"phone_number": phone, "otp": wrong, "is_totp": false},
		})
		assertErrorContains(t, res, "otp")
	}
	locked := rawGraphQL(t, jar, baseURL, verifyOTPMutation, map[string]any{
		"params": map[string]any{"phone_number": phone, "otp": wrong, "is_totp": false},
	})
	assertErrorContains(t, locked, "too many failed attempts")

	stillLocked := rawGraphQL(t, jar, baseURL, verifyOTPMutation, map[string]any{
		"params": map[string]any{"phone_number": phone, "otp": correct, "is_totp": false},
	})
	assertErrorContains(t, stillLocked, "too many failed attempts")
}

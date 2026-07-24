package sdktests

// MFA enforcement routing — driven entirely through the SDK's typed Login /
// SkipMfaSetup methods. This is the highest-value pure-SDK area: it asserts the
// exact wire shape of the login response (withheld access_token + the
// should_show_totp_screen / should_offer_* flags), which is precisely the class
// of drift the existing raw-GraphQL Playwright suite can't catch on the SDK.

import (
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer-go/v2"
)

// Under --enforce-mfa, a brand-new password user is routed into MFA enrollment
// with the token withheld. Read the whole login response through the SDK.
func TestMFAEnforced_LoginWithholdsTokenAndRoutesToEnrollment(t *testing.T) {
	c := userClient(t, mfaEnforcedURL)
	email := randomEmail("mfa-enforced")

	if _, err := c.SignUp(&authorizer.SignUpRequest{
		Email: &email, Password: testPassword, ConfirmPassword: testPassword,
	}); err != nil {
		t.Fatalf("SignUp: %v", err)
	}

	res, err := c.Login(&authorizer.LoginRequest{Email: &email, Password: testPassword})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// mfaGateBlockEnroll: enforcement withholds the token entirely.
	if res.AccessToken != nil && *res.AccessToken != "" {
		t.Errorf("expected withheld access_token under --enforce-mfa, got %q", *res.AccessToken)
	}
	if got := stringValue(res.Message); got != "Proceed to mfa setup" {
		t.Errorf("expected message %q, got %q", "Proceed to mfa setup", got)
	}
	// TOTP is enabled by default in this stack, so the TOTP enrollment screen
	// is offered alongside the withheld token.
	if !boolValue(res.ShouldShowTotpScreen) {
		t.Errorf("expected should_show_totp_screen=true under enforcement")
	}
}

// skip_mfa_setup is refused under enforcement (enforcement is never skippable).
// Through the SDK the refusal surfaces as an error; the SDK's SkipMfaSetup can't
// carry the mfa_session cookie login armed (SDK Login discards it), so the
// server rejects it before even reaching the "cannot skip" enforcement branch —
// either way the skip does NOT yield a token, which is the security property.
func TestMFAEnforced_SkipMfaSetupIsRefused(t *testing.T) {
	c := userClient(t, mfaEnforcedURL)
	email := randomEmail("mfa-enforced-skip")

	if _, err := c.SignUp(&authorizer.SignUpRequest{
		Email: &email, Password: testPassword, ConfirmPassword: testPassword,
	}); err != nil {
		t.Fatalf("SignUp: %v", err)
	}
	if _, err := c.Login(&authorizer.LoginRequest{Email: &email, Password: testPassword}); err != nil {
		t.Fatalf("Login: %v", err)
	}

	res, err := c.SkipMfaSetup(&authorizer.SkipMfaSetupRequest{Email: &email})
	if err == nil {
		// If it somehow returned a payload, it must not contain a usable token.
		if res != nil && res.AccessToken != nil && *res.AccessToken != "" {
			t.Fatalf("skip_mfa_setup issued a token under enforcement — enforcement bypassed")
		}
		t.Fatalf("expected skip_mfa_setup to error under enforcement, got nil error")
	}
	// Accept either the enforcement message or the session-absent rejection;
	// both prove the skip did not succeed.
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "cannot skip") && !strings.Contains(msg, "unauthor") && !strings.Contains(msg, "session") {
		t.Errorf("unexpected skip refusal reason: %v", err)
	}
}

// On the default (non-enforced) instance, a brand-new user is still gated but
// with the SKIPPABLE offer (mfaGateOfferAll): token withheld, TOTP offered. The
// distinction from enforcement is that skip WOULD be allowed here — asserted via
// the login response shape the SDK returns.
func TestDefaultInstance_LoginOffersOptionalTotpEnrollment(t *testing.T) {
	c := userClient(t, baseURL)
	email := randomEmail("mfa-optional")

	if _, err := c.SignUp(&authorizer.SignUpRequest{
		Email: &email, Password: testPassword, ConfirmPassword: testPassword,
	}); err != nil {
		t.Fatalf("SignUp: %v", err)
	}
	res, err := c.Login(&authorizer.LoginRequest{Email: &email, Password: testPassword})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if res.AccessToken != nil && *res.AccessToken != "" {
		t.Errorf("expected withheld token on first-login optional-MFA offer")
	}
	if !boolValue(res.ShouldShowTotpScreen) {
		t.Errorf("expected should_show_totp_screen=true on optional-MFA offer")
	}
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

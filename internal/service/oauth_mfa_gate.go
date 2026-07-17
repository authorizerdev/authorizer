// internal/service/oauth_mfa_gate.go
package service

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// EvaluateMFAGateForOAuth is oauth_callback.go's entry point into the same
// gate Login/SignUp/WebauthnLoginVerify use. See interface doc comment on
// Provider.EvaluateMFAGateForOAuth. Also used by the REST VerifyEmailHandler
// (magic-link/email-verification click-through), which needs the same
// gate-then-redirect-with-mfa_required shape - name is a historical
// carryover from its original caller, not OAuth-specific.
//
// Like WebauthnLoginVerify, an OAuth/social login is only one factor
// (something you have — the provider's own session) and does not itself
// satisfy an MFA requirement. Unlike login.go, OAuth has no
// isEmailLogin/isMobileLogin concept to short-circuit into an inline
// "send the OTP now" branch, so a verified Email/SMS-OTP authenticator is
// folded directly into authenticatorVerified here — same enrollment check
// (GetAuthenticatorDetailsByUserId + VerifiedAt) login.go already uses to
// decide whether to take its own email/SMS branches. The frontend resolves
// a mfaGateBlockVerify+email_otp/sms_otp hint via ResendOTP, which sends
// the code and sets the MFA session cookie for the verify step.
func (p *provider) EvaluateMFAGateForOAuth(ctx context.Context, meta RequestMetadata, side *ResponseSideEffects, user *schemas.User) (bool, string, error) {
	if user.MFALockedAt != nil {
		return false, "", FailedPrecondition("your account's multi-factor authentication is locked; contact your administrator to regain access")
	}

	webauthnCreds, _ := p.StorageProvider.ListWebauthnCredentialsByUserID(ctx, user.ID)
	totpAuthenticator, _ := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
	totpVerified := totpAuthenticator != nil && totpAuthenticator.VerifiedAt != nil
	emailOTPAuthenticator, _ := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyEmailOTPAuthenticator)
	emailOTPVerified := emailOTPAuthenticator != nil && emailOTPAuthenticator.VerifiedAt != nil
	smsOTPAuthenticator, _ := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeySMSOTPAuthenticator)
	smsOTPVerified := smsOTPAuthenticator != nil && smsOTPAuthenticator.VerifiedAt != nil
	authenticatorVerified := totpVerified || len(webauthnCreds) > 0 || emailOTPVerified || smsOTPVerified

	gate := resolveMFAGate(effectiveMFAEnabled(p.Config, user), p.Config.EnforceMFA, authenticatorVerified, user.HasSkippedMFASetupAt != nil)
	switch gate {
	case mfaGateNone, mfaGateSkippedSetup:
		return false, "", nil
	}

	expiresAt := time.Now().Add(3 * time.Minute).Unix()
	if err := p.setMFASession(meta, side, user.ID, expiresAt); err != nil {
		return false, "", err
	}

	methods := []string{}
	switch gate {
	case mfaGateBlockVerify:
		if totpVerified {
			methods = append(methods, constants.EnvKeyTOTPAuthenticator)
		}
		if len(webauthnCreds) > 0 {
			methods = append(methods, constants.AuthRecipeMethodWebauthn)
		}
		if emailOTPVerified {
			methods = append(methods, constants.EnvKeyEmailOTPAuthenticator)
		}
		if smsOTPVerified {
			methods = append(methods, constants.EnvKeySMSOTPAuthenticator)
		}
	case mfaGateBlockEnroll, mfaGateOfferAll:
		if p.Config.EnableTOTPLogin {
			methods = append(methods, constants.EnvKeyTOTPAuthenticator)
		}
		if p.Config.EnableWebauthnMFA {
			methods = append(methods, constants.AuthRecipeMethodWebauthn)
		}
		if p.Config.EnableEmailOTP && p.Config.IsEmailServiceEnabled {
			methods = append(methods, constants.EnvKeyEmailOTPAuthenticator)
		}
		if p.Config.EnableSMSOTP && p.Config.IsSMSServiceEnabled {
			methods = append(methods, constants.EnvKeySMSOTPAuthenticator)
		}
	}
	q := url.Values{}
	q.Set("mfa_required", "1")
	q.Set("mfa_methods", strings.Join(methods, ","))
	// Deliberately no email/phone_number here: OAuth's continuation calls
	// (verify_otp/skip_mfa_setup/lock_mfa) resolve the account from the MFA
	// session cookie alone when no identifier is supplied — see
	// MemoryStoreProvider.GetMfaSessionOwner. Putting PII in a redirect URL
	// risks referrer leakage to third-party scripts, proxy/CDN access logs,
	// and browser history — avoided entirely by not needing it here.
	return true, q.Encode(), nil
}

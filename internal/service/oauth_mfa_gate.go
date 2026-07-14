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
// Provider.EvaluateMFAGateForOAuth.
//
// Like WebauthnLoginVerify, an OAuth/social login is only one factor
// (something you have — the provider's own session) and does not itself
// satisfy an MFA requirement, so authenticatorVerified below only considers
// TOTP + WebAuthn (not email/SMS OTP, which login.go only offers for a
// password-based primary login that has an associated email/phone to send
// a code to).
func (p *provider) EvaluateMFAGateForOAuth(ctx context.Context, meta RequestMetadata, side *ResponseSideEffects, user *schemas.User) (bool, string, error) {
	if user.MFALockedAt != nil {
		return false, "", FailedPrecondition("your account's multi-factor authentication is locked; contact your administrator to regain access")
	}

	webauthnCreds, _ := p.StorageProvider.ListWebauthnCredentialsByUserID(ctx, user.ID)
	totpAuthenticator, _ := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
	totpVerified := totpAuthenticator != nil && totpAuthenticator.VerifiedAt != nil
	authenticatorVerified := totpVerified || len(webauthnCreds) > 0

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
	case mfaGateBlockEnroll, mfaGateOfferAll:
		methods = append(methods, constants.EnvKeyTOTPAuthenticator)
		if p.Config.EnableWebauthnMFA {
			methods = append(methods, constants.AuthRecipeMethodWebauthn)
		}
	}
	q := url.Values{}
	q.Set("mfa_required", "1")
	q.Set("mfa_methods", strings.Join(methods, ","))
	return true, q.Encode(), nil
}

// internal/service/mfa_gate.go
package service

import (
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// mfaGateDecision is what login.go should do once it knows a user has MFA
// available. See resolveMFAGate for the truth table.
type mfaGateDecision int

const (
	// mfaGateNone: user has no MFA to worry about. Issue the token normally.
	mfaGateNone mfaGateDecision = iota
	// mfaGateBlockVerify: user has a verified/completed MFA method already.
	// Withhold the token until they verify it. Never skippable — this is the
	// user's own opted-in second factor.
	mfaGateBlockVerify
	// mfaGateBlockEnroll: MFA is org-enforced and this user hasn't finished
	// enrollment yet. Withhold the token until enrollment is completed.
	// Never skippable.
	mfaGateBlockEnroll
	// mfaGateOfferAll: MFA is available but not enforced, the user hasn't
	// enrolled, and they've never skipped before. Token is WITHHELD (same
	// group as mfaGateBlockVerify/mfaGateBlockEnroll) until the user
	// completes one method or explicitly calls skip_mfa_setup — both of
	// which authenticate via the MFA session cookie this decision triggers,
	// not a bearer token, since none has been issued yet.
	mfaGateOfferAll
	// mfaGateSkippedSetup: same as mfaGateOfferAll but the user has already
	// chosen Skip in the past. Issue the token, don't nag.
	mfaGateSkippedSetup
)

// resolveMFAGate decides what login.go does for a user whose
// IsMultiFactorAuthEnabled flag might be set. Only called when the caller
// has already confirmed MFA is available on this server at all
// (Config.EnableMFA) — see login.go call sites.
//
//   - userMFAEnabled: schemas.User.IsMultiFactorAuthEnabled
//   - enforceMFA: Config.EnforceMFA (org-wide mandate — absolute, never
//     bypassed by hasSkippedSetup)
//   - authenticatorVerified: true when the user has a completed/verified MFA
//     method already (e.g. a verified TOTP authenticator) — their own opted-in
//     second factor, always required once true, regardless of enforceMFA or
//     hasSkippedSetup
//   - hasSkippedSetup: schemas.User.HasSkippedMFASetupAt != nil
func resolveMFAGate(userMFAEnabled, enforceMFA, authenticatorVerified, hasSkippedSetup bool) mfaGateDecision {
	if !userMFAEnabled {
		return mfaGateNone
	}
	if authenticatorVerified {
		// The user's own completed second factor. Always required, never
		// skippable, regardless of current enforcement policy.
		return mfaGateBlockVerify
	}
	if enforceMFA {
		return mfaGateBlockEnroll
	}
	if hasSkippedSetup {
		return mfaGateSkippedSetup
	}
	return mfaGateOfferAll
}

// effectiveMFAEnabled reports whether MFA applies to this user right now.
// Never persisted — recomputed from current config plus the user's own
// explicit choice, if any. Replaces the old signup-time default-write and
// login-time backfill: IsMultiFactorAuthEnabled is non-nil ONLY when a
// caller explicitly set it (SignUp params, _update_user params) — everyone
// else follows whatever cfg.EnableMFA currently is, live, every call.
func effectiveMFAEnabled(cfg *config.Config, user *schemas.User) bool {
	if user.IsMultiFactorAuthEnabled != nil {
		return refs.BoolValue(user.IsMultiFactorAuthEnabled)
	}
	return cfg.EnableMFA
}

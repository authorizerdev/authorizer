// internal/service/mfa_gate.go
package service

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
	// mfaGateOfferSetup: MFA is available but not enforced, the user hasn't
	// enrolled, and they've never skipped before. Issue the token now AND
	// tell the frontend to offer (not force) MFA setup.
	mfaGateOfferSetup
	// mfaGateSkippedSetup: same as mfaGateOfferSetup but the user has already
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
	return mfaGateOfferSetup
}

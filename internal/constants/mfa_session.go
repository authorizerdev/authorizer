package constants

// MFA session purposes tag how a short-lived MFA session (cookie + memory-store
// row keyed by user ID) was obtained, so a consumer that acts on the strength
// of a bare session can tell a first-factor-verified caller from one who only
// triggered an OTP send.
const (
	// MFASessionPurposeVerified means the caller already completed a first
	// factor (password/passkey/social login) for this exact user, or this is
	// the user's own just-created account.
	MFASessionPurposeVerified = "verified"
	// MFASessionPurposeChallenge means the caller only proved they can trigger
	// an OTP send to this email/phone. NOT sufficient to skip MFA setup or lock
	// the account.
	MFASessionPurposeChallenge = "challenge"
)

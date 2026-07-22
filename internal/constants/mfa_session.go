package constants

// MFA session purposes tag how a short-lived MFA session (cookie + memory-store
// row keyed by user ID) was obtained, so a consumer that acts on the strength
// of a bare session can tell a first-factor-verified caller from one who only
// triggered an OTP send, and can tell WHICH pending flow a bare OTP-send
// session belongs to.
const (
	// MFASessionPurposeVerified means the caller already completed a first
	// factor (password/passkey/social login) for this exact user, or this is
	// the user's own just-created account.
	MFASessionPurposeVerified = "verified"
	// MFASessionPurposeChallenge means the caller only proved they can trigger
	// a login/signup/MFA OTP send to this email/phone. NOT sufficient to skip
	// MFA setup or lock the account. Consumed exclusively by VerifyOTP.
	MFASessionPurposeChallenge = "challenge"
	// MFASessionPurposePasswordReset means the caller only proved they can
	// trigger a password-reset OTP send to this phone number (ForgotPassword's
	// mobile leg). It authorizes ONLY a password change via ResetPassword — it
	// must never be redeemable for an access token via VerifyOTP, and the
	// Challenge/Verified purposes above must never be accepted by
	// ResetPassword's OTP path. Keeping these mutually exclusive is what closes
	// the MFA-downgrade path where a password-reset OTP was redeemable for a
	// full token instead of only a password change.
	MFASessionPurposePasswordReset = "password_reset"
)

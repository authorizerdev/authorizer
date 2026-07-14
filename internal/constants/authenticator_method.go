package constants

// Authenticators Methods
const (
	// EnvKeyTOTPAuthenticator key for env variable TOTP
	EnvKeyTOTPAuthenticator = "totp"
	// EnvKeyEmailOTPAuthenticator key for email OTP used as an MFA factor.
	EnvKeyEmailOTPAuthenticator = "email_otp"
	// EnvKeySMSOTPAuthenticator key for SMS OTP used as an MFA factor.
	EnvKeySMSOTPAuthenticator = "sms_otp"
)

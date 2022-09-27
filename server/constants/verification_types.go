package constants

const (
	// VerificationTypeBasicAuthSignup is the basic_auth_signup verification type
	VerificationTypeBasicAuthSignup = "basic_auth_signup"
	// VerificationTypeMagicLinkLogin is the magic_link_login verification type
	VerificationTypeMagicLinkLogin = "magic_link_login"
	// VerificationTypeUpdateEmail is the update_email verification type
	VerificationTypeUpdateEmail = "update_email"
	// VerificationTypeForgotPassword is the forgot_password verification type
	VerificationTypeForgotPassword = "forgot_password"
	// VerificationTypeInviteMember is the invite_member verification type
	VerificationTypeInviteMember = "invite_member"
	// VerificationTypeOTP is the otp verification type
	VerificationTypeOTP = "verify_otp"
)

var (
	// VerificationTypes is slice of all verification types
	VerificationTypes = []string{
		VerificationTypeBasicAuthSignup,
		VerificationTypeMagicLinkLogin,
		VerificationTypeUpdateEmail,
		VerificationTypeForgotPassword,
		VerificationTypeInviteMember,
	}
)

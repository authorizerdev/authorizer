package service

import (
	"github.com/golang-jwt/jwt/v4"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

var (
	// verificationPurposesResetPassword is the only purpose ResetPassword's
	// token leg accepts. ForgotPassword is the sole minter of it.
	verificationPurposesResetPassword = []string{
		constants.VerificationTypeForgotPassword,
	}
	// verificationPurposesVerifyEmail is the email-verification family the
	// verify-email endpoints legitimately complete: signup, magic-link login, an
	// admin invite, and an email-change confirmation. Notably absent is
	// forgot_password — redeeming one here would hand out a full session on a
	// token minted for a password reset.
	verificationPurposesVerifyEmail = []string{
		constants.VerificationTypeBasicAuthSignup,
		constants.VerificationTypeMagicLinkLogin,
		constants.VerificationTypeUpdateEmail,
		constants.VerificationTypeInviteMember,
	}
)

// verificationPurposeAllowed reports whether a verification request and the
// signed `token_type` claim of the token that produced it were both issued for
// one of the purposes the calling endpoint actually serves.
//
// Tokens for every flow — signup, magic link, invite, email change, forgot
// password — share one `verification_requests` table keyed by the token string
// alone, so GetVerificationRequestByToken will happily hand a magic-link row to
// ResetPassword. Without this gate any leaked one-time login link (referer
// leakage, proxy logs, a shared URL) is redeemable for a password change, which
// also appends basic_auth to the account's signup methods — a scoped, one-shot
// capability escalated into durable account takeover. The reverse (a
// forgot-password token redeemed at VerifyEmail for a full session) works too.
//
// Both the stored Identifier and the signed claim are checked. Neither alone is
// sufficient: the DB row is selected by the attacker-supplied token, and the
// claim alone would reject the invite flow, which deliberately signs
// `invite_member` while storing `magic_link_login` as the identifier when magic
// link login is enabled (admin_access.go).
func verificationPurposeAllowed(req *schemas.VerificationRequest, claim jwt.MapClaims, allowed []string) bool {
	if req == nil {
		return false
	}
	if !utils.StringSliceContains(allowed, req.Identifier) {
		return false
	}
	tokenType, _ := claim["token_type"].(string)
	return utils.StringSliceContains(allowed, tokenType)
}

// IsVerifyEmailPurpose is the exported form of the check for the email
// verification family, for callers outside this package.
//
// Exported because there are TWO implementations of email verification over the
// same token table — the GraphQL mutation (VerifyEmail, verify_email.go) and the
// REST handler behind GET /verify_email, which is the URL every verification and
// magic-link mail actually points at (utils.GetEmailVerificationURL). They share
// no code, so a gate applied to only one of them is not a gate: a forgot-password
// token rejected by the mutation stays redeemable at the REST route for a full
// session. Both must call this.
func IsVerifyEmailPurpose(req *schemas.VerificationRequest, claim jwt.MapClaims) bool {
	return verificationPurposeAllowed(req, claim, verificationPurposesVerifyEmail)
}

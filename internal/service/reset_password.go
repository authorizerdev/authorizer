package service

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// ResetPassword resets a user's password using either an email verification
// token or a mobile OTP. Transport-agnostic port of
// graphqlProvider.ResetPassword.
//
// Permissions: none.
func (p *provider) ResetPassword(ctx context.Context, meta RequestMetadata, params *model.ResetPasswordRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "ResetPassword").Logger()
	side := &ResponseSideEffects{}

	verifyingToken := refs.StringValue(params.Token)
	otp := refs.StringValue(params.Otp)
	if verifyingToken == "" && otp == "" {
		log.Debug().Msg("Token or otp is required")
		return nil, nil, InvalidArgument(`token or otp is required`)
	}
	isTokenVerification := verifyingToken != ""
	isOtpVerification := otp != ""
	if isOtpVerification && refs.StringValue(params.PhoneNumber) == "" {
		log.Debug().Msg("Phone number is required")
		return nil, nil, InvalidArgument(`phone number is required`)
	}
	isBasicAuthEnabled := p.Config.EnableBasicAuthentication
	isMobileBasicAuthEnabled := p.Config.EnableMobileBasicAuthentication
	if isTokenVerification && !isBasicAuthEnabled {
		log.Debug().Msg("Basic authentication is disabled")
		return nil, nil, FailedPrecondition(`basic authentication is disabled for this instance`)
	}
	if isOtpVerification && !isMobileBasicAuthEnabled {
		log.Debug().Msg("Mobile basic authentication is disabled")
		return nil, nil, FailedPrecondition(`mobile basic authentication is disabled for this instance`)
	}
	email := ""
	phoneNumber := refs.StringValue(params.PhoneNumber)
	var user *schemas.User
	var verificationRequest *schemas.VerificationRequest
	var otpRequest *schemas.OTP
	var err error
	if isTokenVerification {
		verificationRequest, err = p.StorageProvider.GetVerificationRequestByToken(ctx, verifyingToken)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get verification request")
			return nil, nil, InvalidArgument(`invalid token`)
		}
		// verify if token exists in db
		hostname := meta.HostURL
		claim, err := p.TokenProvider.ParseJWTToken(verifyingToken)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to parse token")
			return nil, nil, InvalidArgument(`invalid token`)
		}

		if ok, err := p.TokenProvider.ValidateJWTClaims(claim, &token.AuthTokenConfig{
			HostName: hostname,
			Nonce:    verificationRequest.Nonce,
			User: &schemas.User{
				ID:    "",
				Email: refs.NewStringRef(verificationRequest.Email),
			},
		}); !ok || err != nil {
			log.Debug().Err(err).Msg("Failed to validate jwt claims")
			return nil, nil, InvalidArgument(`invalid token`)
		}
		email = claim["sub"].(string)
		user, err = p.StorageProvider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user")
			return nil, nil, err
		}
	}
	if isOtpVerification {
		// cookie.GetMfaSession reads the MFA cookie off the inbound request;
		// synthesize a minimal gin.Context wrapping it for both gin and
		// non-gin transports.
		gc := &gin.Context{Request: meta.Request}
		mfaSession, err := cookie.GetMfaSession(gc)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get mfa session cookie")
			return nil, nil, Unauthenticated(`invalid session`)
		}
		// Get user by phone number
		user, err = p.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by phone number")
			return nil, nil, NotFound(`user not found`)
		}
		// Only a password_reset-purpose session (minted exclusively by
		// ForgotPassword's mobile leg) may complete a password change here.
		// Verified/Challenge sessions from unrelated flows (login, signup,
		// resend-OTP) must not be redeemable for a password change.
		purpose, err := p.MemoryStoreProvider.GetMfaSession(user.ID, mfaSession)
		if err != nil || purpose != constants.MFASessionPurposePasswordReset {
			log.Debug().Err(err).Msg("Failed to get mfa session")
			return nil, nil, Unauthenticated(`invalid session`)
		}
		otpRequest, err = p.StorageProvider.GetOTPByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get otp request by phone number")
			return nil, nil, InvalidArgument(`invalid otp`)
		}
		// OTPs are stored as HMAC-SHA256 digests; we deliberately do NOT
		// fall back to literal equality so the stored digest cannot be
		// replayed as a credential by anyone with DB read access.
		if !crypto.VerifyOTPHash(otp, otpRequest.Otp, p.Config.JWTSecret) {
			log.Debug().Msg("Failed to verify otp request: Incorrect value")
			return nil, nil, InvalidArgument(`invalid otp`)
		}
		if otpRequest.ExpiresAt < time.Now().Unix() {
			log.Debug().Msg("OTP has expired")
			return nil, nil, InvalidArgument("otp expired")
		}
	}
	if params.Password != params.ConfirmPassword {
		log.Debug().Msg("Passwords do not match")
		return nil, nil, InvalidArgument(`passwords don't match`)
	}
	if err := validators.IsValidPassword(params.Password, !p.Config.EnableStrongPassword); err != nil {
		log.Debug().Msg("Invalid password")
		return nil, nil, InvalidArgument(err.Error())
	}
	log = log.With().Str("email", email).Str("phone_number", phoneNumber).Logger()
	password, _ := crypto.EncryptPassword(params.Password)
	user.Password = &password
	signupMethod := user.SignupMethods
	if !strings.Contains(signupMethod, constants.AuthRecipeMethodBasicAuth) && isTokenVerification {
		signupMethod = signupMethod + "," + constants.AuthRecipeMethodBasicAuth
		// helpful if user has not signed up with basic auth
		if user.EmailVerifiedAt == nil {
			now := time.Now().Unix()
			user.EmailVerifiedAt = &now
		}
	}
	if !strings.Contains(signupMethod, constants.AuthRecipeMethodMobileOTP) && isOtpVerification {
		signupMethod = signupMethod + "," + constants.AuthRecipeMethodMobileOTP
		// helpful if user has not signed up with basic auth
		if user.PhoneNumberVerifiedAt == nil {
			now := time.Now().Unix()
			user.PhoneNumberVerifiedAt = &now
		}
	}
	user.SignupMethods = signupMethod
	isMFAEnforced := p.Config.EnforceMFA
	if isMFAEnforced {
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	}
	_, err = p.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, nil, err
	}
	if isTokenVerification {
		// delete from verification table
		err = p.StorageProvider.DeleteVerificationRequest(ctx, verificationRequest)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to delete verification request")
			return nil, nil, err
		}
	}
	if isOtpVerification {
		// delete from otp table
		err = p.StorageProvider.DeleteOTP(ctx, otpRequest)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to delete otp request")
			return nil, nil, err
		}
	}
	// A password reset must terminate every pre-existing session and refresh
	// token before the caller is told it succeeded: the whole point is to
	// lock out anyone who held the old credential. Synchronous (not
	// fire-and-forget like UpdateProfile/DeactivateAccount) to close the
	// window where an attacker's pre-existing token could still be used
	// between the response going out and the goroutine actually running.
	// Logged rather than silently ignored: a memory-store fault here means
	// old sessions may still be live even though the reset "succeeded".
	if err := p.MemoryStoreProvider.DeleteAllUserSessions(user.ID); err != nil {
		log.Debug().Err(err).Msg("Failed to revoke existing sessions after password reset")
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditPasswordResetEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	metrics.RecordAuthEvent(metrics.EventResetPwd, metrics.StatusSuccess)
	return &model.Response{
		Message: `Password updated successfully.`,
	}, side, nil
}

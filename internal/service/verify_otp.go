package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// VerifyOTP verifies a one-time passcode (email/SMS OTP, TOTP, or recovery
// code) for a pending MFA session and, on success, issues an auth token.
// Transport-agnostic port of graphqlProvider.VerifyOTP.
//
// Permissions: none — completes an in-progress authentication identified by the
// MFA session cookie.
func (p *provider) VerifyOTP(ctx context.Context, meta RequestMetadata, params *model.VerifyOTPRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "VerifyOTP").Logger()
	side := &ResponseSideEffects{}

	// The MFA session lives in a request cookie; cookie.GetMfaSession still
	// reads from a gin.Context, so wrap the inbound *http.Request.
	gc := &gin.Context{Request: meta.Request}
	mfaSession, err := cookie.GetMfaSession(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get mfa session")
		return nil, nil, Unauthenticated(`invalid session`)
	}

	email := strings.TrimSpace(refs.StringValue(params.Email))
	phoneNumber := strings.TrimSpace(refs.StringValue(params.PhoneNumber))
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, nil, InvalidArgument(`email or phone number is required`)
	}
	isEmailVerification := email != ""
	isMobileVerification := phoneNumber != ""
	log = log.With().Str("email", email).Str("phone_number", phoneNumber).Logger()
	// Get user by email or phone number
	var user *schemas.User
	if isEmailVerification {
		user, err = p.StorageProvider.GetUserByEmail(ctx, refs.StringValue(params.Email))
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by email")
		}
	} else {
		user, err = p.StorageProvider.GetUserByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by phone number")
		}
	}
	if user == nil || err != nil {
		log.Debug().Err(err).Msg("User not found")
		return nil, nil, NotFound("invalid verification request")
	}

	if user.RevokedTimestamp != nil {
		log.Debug().Msg("User access has been revoked")
		return nil, nil, FailedPrecondition("invalid verification request")
	}

	// Verify OTP based on TOPT or OTP
	if refs.BoolValue(params.IsTotp) {
		status, err := p.AuthenticatorProvider.Validate(ctx, params.Otp, user.ID)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to validate passcode")
			return nil, nil, errors.New("error while validating passcode")
		}
		if !status {
			log.Debug().Msg("Failed to verify otp request: Incorrect value")
			log.Info().Msg("Checking if otp is recovery code")
			// Check if otp is recovery code
			isValidRecoveryCode, err := p.AuthenticatorProvider.ValidateRecoveryCode(ctx, params.Otp, user.ID)
			if err != nil {
				log.Debug().Err(err).Msg("Failed to validate recovery code")
				return nil, nil, errors.New("error while validating recovery code")
			}
			if !isValidRecoveryCode {
				log.Debug().Msg("Failed to verify otp request: Incorrect value")
				return nil, nil, InvalidArgument(`invalid otp`)
			}
		}
	} else {
		var otp *schemas.OTP
		if isEmailVerification {
			otp, err = p.StorageProvider.GetOTPByEmail(ctx, refs.StringValue(params.Email))
			if err != nil {
				log.Debug().Err(err).Msg("Failed to get otp request for email")
			}
		} else {
			otp, err = p.StorageProvider.GetOTPByPhoneNumber(ctx, refs.StringValue(params.PhoneNumber))
			if err != nil {
				log.Debug().Err(err).Msg("Failed to get otp request for phone number")
			}
		}
		if otp == nil {
			log.Debug().Msg("OTP not found")
			return nil, nil, NotFound(`OTP not found`)
		}
		// OTPs are stored as HMAC-SHA256 digests so an offline DB dump no
		// longer reveals usable codes. We deliberately do NOT fall back
		// to literal equality — accepting the stored value verbatim
		// would turn the digest itself into a usable credential.
		if !crypto.VerifyOTPHash(params.Otp, otp.Otp, p.Config.JWTSecret) {
			log.Debug().Msg("Failed to verify otp request: OTP mismatch")
			return nil, nil, InvalidArgument(`invalid otp`)
		}
		expiresIn := otp.ExpiresAt - time.Now().Unix()
		if expiresIn < 0 {
			log.Debug().Msg("OTP expired")
			return nil, nil, InvalidArgument("otp expired")
		}
		if err := p.StorageProvider.DeleteOTP(ctx, otp); err != nil {
			log.Debug().Err(err).Msg("Failed to delete otp")
		}
	}

	if _, err := p.MemoryStoreProvider.GetMfaSession(user.ID, mfaSession); err != nil {
		log.Debug().Err(err).Msg("Failed to get mfa session")
		return nil, nil, Unauthenticated(`invalid session`)
	}

	isSignUp := false
	if user.EmailVerifiedAt == nil && isEmailVerification {
		isSignUp = true
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
	}
	if user.PhoneNumberVerifiedAt == nil && isMobileVerification {
		isSignUp = true
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
	}
	if isSignUp {
		user, err = p.StorageProvider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to update user")
			return nil, nil, err
		}
	}
	loginMethod := constants.AuthRecipeMethodBasicAuth
	if isMobileVerification {
		loginMethod = constants.AuthRecipeMethodMobileOTP
	}
	if isEmailVerification {
		p.AuditProvider.LogEvent(audit.Event{
			Action:   constants.AuditEmailVerifiedEvent,
			Protocol: meta.Protocol, ActorID: user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   refs.StringValue(user.Email),
			ResourceType: constants.AuditResourceTypeUser,
			ResourceID:   user.ID,
			IPAddress:    meta.IPAddress,
			UserAgent:    meta.UserAgent,
		})
	} else {
		p.AuditProvider.LogEvent(audit.Event{
			Action:   constants.AuditPhoneVerifiedEvent,
			Protocol: meta.Protocol, ActorID: user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   refs.StringValue(user.Email),
			ResourceType: constants.AuditResourceTypeUser,
			ResourceID:   user.ID,
			IPAddress:    meta.IPAddress,
			UserAgent:    meta.UserAgent,
		})
	}

	res, err := p.issueAuthResponse(ctx, meta, side, user, loginMethod, `OTP verified successfully.`, params.State, isSignUp)
	if err != nil {
		return nil, nil, err
	}
	return res, side, nil
}

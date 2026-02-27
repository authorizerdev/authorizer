package graphql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// ResetPassword is the method to reset password.
// Permissions: none
func (g *graphqlProvider) ResetPassword(ctx context.Context, params *model.ResetPasswordRequest) (*model.Response, error) {
	log := g.Log.With().Str("func", "ResetPassword").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	verifyingToken := refs.StringValue(params.Token)
	otp := refs.StringValue(params.Otp)
	if verifyingToken == "" && otp == "" {
		log.Debug().Msg("Token or otp is required")
		return nil, fmt.Errorf(`token or otp is required`)
	}
	isTokenVerification := verifyingToken != ""
	isOtpVerification := otp != ""
	if isOtpVerification && refs.StringValue(params.PhoneNumber) == "" {
		log.Debug().Msg("Phone number is required")
		return nil, fmt.Errorf(`phone number is required`)
	}
	isBasicAuthEnabled := g.Config.EnableBasicAuthentication
	isMobileBasicAuthEnabled := g.Config.EnableMobileBasicAuthentication
	if isTokenVerification && !isBasicAuthEnabled {
		log.Debug().Msg("Basic authentication is disabled")
		return nil, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	if isOtpVerification && !isMobileBasicAuthEnabled {
		log.Debug().Msg("Mobile basic authentication is disabled")
		return nil, fmt.Errorf(`mobile basic authentication is disabled for this instance`)
	}
	email := ""
	phoneNumber := refs.StringValue(params.PhoneNumber)
	var user *schemas.User
	var verificationRequest *schemas.VerificationRequest
	var otpRequest *schemas.OTP
	if isTokenVerification {
		verificationRequest, err = g.StorageProvider.GetVerificationRequestByToken(ctx, verifyingToken)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get verification request")
			return nil, fmt.Errorf(`invalid token`)
		}
		// verify if token exists in db
		hostname := parsers.GetHost(gc)
		claim, err := g.TokenProvider.ParseJWTToken(verifyingToken)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to parse token")
			return nil, fmt.Errorf(`invalid token`)
		}

		if ok, err := g.TokenProvider.ValidateJWTClaims(claim, &token.AuthTokenConfig{
			HostName: hostname,
			Nonce:    verificationRequest.Nonce,
			User: &schemas.User{
				ID:    "",
				Email: refs.NewStringRef(verificationRequest.Email),
			},
		}); !ok || err != nil {
			log.Debug().Err(err).Msg("Failed to validate jwt claims")
			return nil, fmt.Errorf(`invalid token`)
		}
		email = claim["sub"].(string)
		user, err = g.StorageProvider.GetUserByEmail(ctx, email)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user")
			return nil, err
		}
	}
	if isOtpVerification {
		mfaSession, err := cookie.GetMfaSession(gc)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get otp request by email")
			return nil, fmt.Errorf(`invalid session: %s`, err.Error())
		}
		// Get user by phone number
		user, err = g.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get user by phone number")
			return nil, fmt.Errorf(`user not found`)
		}
		if _, err := g.MemoryStoreProvider.GetMfaSession(user.ID, mfaSession); err != nil {
			log.Debug().Err(err).Msg("Failed to get mfa session")
			return nil, fmt.Errorf(`invalid session: %s`, err.Error())
		}
		otpRequest, err = g.StorageProvider.GetOTPByPhoneNumber(ctx, phoneNumber)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to get otp request by phone number")
			return nil, fmt.Errorf(`invalid otp`)
		}
		if otpRequest.Otp != otp {
			log.Debug().Msg("Failed to verify otp request: Incorrect value")
			return nil, fmt.Errorf(`invalid otp`)
		}
	}
	if params.Password != params.ConfirmPassword {
		log.Debug().Msg("Passwords do not match")
		return nil, fmt.Errorf(`passwords don't match`)
	}
	if err := validators.IsValidPassword(params.Password, !g.Config.EnableStrongPassword); err != nil {
		log.Debug().Msg("Invalid password")
		return nil, err
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
	isMFAEnforced := g.Config.EnforceMFA
	if isMFAEnforced {
		user.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
	}
	_, err = g.StorageProvider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to update user")
		return nil, err
	}
	if isTokenVerification {
		// delete from verification table
		err = g.StorageProvider.DeleteVerificationRequest(ctx, verificationRequest)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to delete verification request")
			return nil, err
		}
	}
	if isOtpVerification {
		// delete from otp table
		err = g.StorageProvider.DeleteOTP(ctx, otpRequest)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to delete otp request")
			return nil, err
		}
	}
	return &model.Response{
		Message: `Password updated successfully.`,
	}, nil
}

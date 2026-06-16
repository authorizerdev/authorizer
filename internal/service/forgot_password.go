package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// genericForgotPasswordMessage is returned on every non-error path (and on the
// account-not-found / account-revoked paths) so a caller cannot probe whether
// an account exists.
const genericForgotPasswordMessage = `If an account exists for this email, a password reset link has been sent. Please check your inbox. If you don't receive it within a few minutes, double-check the email address for typos.`

// ForgotPassword issues a password-reset verification token (email) or OTP
// (SMS). Transport-agnostic port of graphqlProvider.ForgotPassword.
//
// Permissions: none.
func (p *provider) ForgotPassword(ctx context.Context, meta RequestMetadata, params *model.ForgotPasswordRequest) (*model.ForgotPasswordResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "ForgotPassword").Logger()
	side := &ResponseSideEffects{}
	isBasicAuthEnabled := p.Config.EnableBasicAuthentication
	isEmailVerificationEnabled := p.Config.EnableEmailVerification
	isMobileBasicAuthEnabled := p.Config.EnableMobileBasicAuthentication
	isMobileVerificationEnabled := p.Config.EnablePhoneVerification
	email := refs.StringValue(params.Email)
	phoneNumber := refs.StringValue(params.PhoneNumber)
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, nil, InvalidArgument("email or phone number is required")
	}
	log = log.With().Str("email", email).Str("phoneNumber", phoneNumber).Logger()
	isEmailLogin := email != ""
	isMobileLogin := phoneNumber != ""
	if !isBasicAuthEnabled && isEmailLogin && isEmailVerificationEnabled {
		log.Debug().Msgf("Basic authentication is disabled.")
		return nil, nil, FailedPrecondition("basic authentication is disabled for this instance")
	}
	if !isMobileBasicAuthEnabled && isMobileLogin && isMobileVerificationEnabled {
		log.Debug().Msgf("Mobile basic authentication is disabled.")
		return nil, nil, FailedPrecondition("mobile basic authentication is disabled for this instance")
	}
	var user *schemas.User
	var err error
	if isEmailLogin {
		user, err = p.StorageProvider.GetUserByEmail(ctx, email)
		log.Debug().Err(err).Msg("Failed to get user by email")
	} else {
		user, err = p.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		log.Debug().Err(err).Msg("Failed to get user by phone number")
	}
	if err != nil {
		// Do not reveal whether the account exists. Return the same generic
		// "we sent the email if it exists" response that a successful path
		// returns. The real reason is logged at debug level.
		log.Debug().Err(err).Str("reason", "user_not_found").Msg("forgot password silently dropped")
		metrics.RecordAuthEvent(metrics.EventForgotPwd, metrics.StatusFailure)
		return &model.ForgotPasswordResponse{Message: genericForgotPasswordMessage}, nil, nil
	}
	hostname := meta.HostURL
	_, nonceHash, err := utils.GenerateNonce()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to generate nonce")
		return nil, nil, err
	}
	if user.RevokedTimestamp != nil {
		log.Debug().Str("reason", "account_revoked").Msg("forgot password silently dropped")
		metrics.RecordAuthEvent(metrics.EventForgotPwd, metrics.StatusFailure)
		return &model.ForgotPasswordResponse{Message: genericForgotPasswordMessage}, nil, nil
	}
	if isEmailLogin {
		redirectURI := ""
		// give higher preference to params redirect uri
		if strings.TrimSpace(refs.StringValue(params.RedirectURI)) != "" {
			redirectURI = refs.StringValue(params.RedirectURI)
			if !validators.IsValidRedirectURI(redirectURI, p.Config.AllowedOrigins, hostname) {
				log.Debug().Msg("Invalid redirect URI")
				return nil, nil, InvalidArgument("invalid redirect URI")
			}
		} else {
			redirectURI = p.Config.ResetPasswordURL
			if redirectURI == "" {
				log.Debug().Msg("Failed to get reset password url")
				redirectURI = hostname + "/app/reset-password"
			}
		}

		verificationToken, err := p.TokenProvider.CreateVerificationToken(&token.AuthTokenConfig{
			LoginMethod: constants.AuthRecipeMethodBasicAuth,
			Nonce:       nonceHash,
			User:        user,
			HostName:    hostname,
		}, redirectURI, constants.VerificationTypeForgotPassword)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to create verification token")
			return nil, nil, err
		}
		_, err = p.StorageProvider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
			Token:       verificationToken,
			Identifier:  constants.VerificationTypeForgotPassword,
			ExpiresAt:   time.Now().Add(time.Minute * 30).Unix(),
			Email:       email,
			Nonce:       nonceHash,
			RedirectURI: redirectURI,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Failed to add verification request")
			return nil, nil, err
		}
		// execute it as go routine so that we can reduce the api latency
		go func() {
			_ = p.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeForgotPassword, map[string]any{
				"user":             user.ToMap(),
				"organization":     utils.GetOrganization(p.Config),
				"verification_url": utils.GetForgotPasswordURL(verificationToken, redirectURI),
			})
		}()
		p.AuditProvider.LogEvent(audit.Event{
			Action:   constants.AuditForgotPasswordEvent,
			Protocol: meta.Protocol, ActorID: user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   email,
			ResourceType: constants.AuditResourceTypeUser,
			ResourceID:   user.ID,
			IPAddress:    meta.IPAddress,
			UserAgent:    meta.UserAgent,
		})
		metrics.RecordAuthEvent(metrics.EventForgotPwd, metrics.StatusSuccess)
		return &model.ForgotPasswordResponse{Message: genericForgotPasswordMessage}, nil, nil
	}
	if isMobileLogin {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
		otp, err := utils.GenerateOTP()
		if err != nil {
			log.Debug().Err(err).Msg("Failed to generate OTP")
			return nil, nil, err
		}
		// Store the HMAC digest; otp (plaintext local) is sent via SMS below.
		_, err = p.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			Email:       refs.StringValue(user.Email),
			PhoneNumber: refs.StringValue(user.PhoneNumber),
			Otp:         crypto.HashOTP(otp, p.Config.JWTSecret),
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Debug().Err(err).Msg("Failed to upsert otp")
			return nil, nil, err
		}
		mfaSession := uuid.NewString()
		err = p.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to set mfa session")
			return nil, nil, err
		}
		for _, c := range cookie.BuildMfaSessionCookies(hostname, mfaSession, p.Config.AppCookieSecure) {
			side.AddCookie(c)
		}
		smsBody := strings.Builder{}
		smsBody.WriteString("Your verification code is: ")
		smsBody.WriteString(otp)
		if err := p.SMSProvider.SendSMS(phoneNumber, smsBody.String()); err != nil {
			log.Debug().Err(err).Msg("Failed to send sms")
			// continue
		}
		p.AuditProvider.LogEvent(audit.Event{
			Action:   constants.AuditForgotPasswordEvent,
			Protocol: meta.Protocol, ActorID: user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   refs.StringValue(user.Email),
			ResourceType: constants.AuditResourceTypeUser,
			ResourceID:   user.ID,
			IPAddress:    meta.IPAddress,
			UserAgent:    meta.UserAgent,
		})
		metrics.RecordAuthEvent(metrics.EventForgotPwd, metrics.StatusSuccess)
		return &model.ForgotPasswordResponse{
			Message:                   "Please enter the OTP sent to your phone number and change your password.",
			ShouldShowMobileOtpScreen: refs.NewBoolRef(true),
		}, side, nil
	}
	return nil, nil, FailedPrecondition("email or phone number verification needs to be enabled")
}

// internal/service/otp_mfa_setup.go
package service

import (
	"context"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// EmailOTPMFASetup sends a one-time code to the authenticated caller's own
// email and creates (or refreshes) an unverified email-OTP Authenticator
// row. Permissions: authenticated caller (bearer token) — this is a
// settings-screen "add a second factor" action.
func (p *provider) EmailOTPMFASetup(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "EmailOTPMFASetup").Logger()

	tokenData, err := p.callerTokenData(ctx, meta)
	if err != nil || tokenData == nil || tokenData.UserID == "" {
		log.Debug().Err(err).Msg("Failed to get user id from session or access token")
		return nil, nil, Unauthenticated("unauthorized")
	}

	if !p.Config.EnableEmailOTP || !p.Config.IsEmailServiceEnabled {
		return nil, nil, FailedPrecondition("email OTP is not available on this server")
	}

	user, err := p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by id")
		return nil, nil, err
	}
	email := strings.TrimSpace(refs.StringValue(user.Email))
	if email == "" {
		return nil, nil, FailedPrecondition("account has no email address to send an OTP to")
	}

	expiresAt := time.Now().Add(1 * time.Minute).Unix()
	otpData, err := p.generateAndStoreOTP(ctx, user, expiresAt)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to generate otp")
		return nil, nil, err
	}

	if err := p.upsertUnverifiedAuthenticator(ctx, user.ID, constants.EnvKeyEmailOTPAuthenticator); err != nil {
		log.Debug().Err(err).Msg("Failed to record pending enrollment")
		return nil, nil, err
	}

	go func() {
		if err := p.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]any{
			"user":         user.ToMap(),
			"organization": utils.GetOrganization(p.Config),
			"otp":          otpData.Otp,
		}); err != nil {
			log.Debug().Msg("Failed to send otp email")
		}
	}()

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditMFAEnabledEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   email,
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{Message: "Check your email for the verification code"}, nil, nil
}

// SMSOTPMFASetup is EmailOTPMFASetup's SMS twin.
func (p *provider) SMSOTPMFASetup(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "SMSOTPMFASetup").Logger()

	tokenData, err := p.callerTokenData(ctx, meta)
	if err != nil || tokenData == nil || tokenData.UserID == "" {
		log.Debug().Err(err).Msg("Failed to get user id from session or access token")
		return nil, nil, Unauthenticated("unauthorized")
	}

	if !p.Config.EnableSMSOTP || !p.Config.IsSMSServiceEnabled {
		return nil, nil, FailedPrecondition("SMS OTP is not available on this server")
	}

	user, err := p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by id")
		return nil, nil, err
	}
	phone := strings.TrimSpace(refs.StringValue(user.PhoneNumber))
	if phone == "" {
		return nil, nil, FailedPrecondition("account has no phone number to send an OTP to")
	}

	expiresAt := time.Now().Add(1 * time.Minute).Unix()
	otpData, err := p.generateAndStoreOTP(ctx, user, expiresAt)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to generate otp")
		return nil, nil, err
	}

	if err := p.upsertUnverifiedAuthenticator(ctx, user.ID, constants.EnvKeySMSOTPAuthenticator); err != nil {
		log.Debug().Err(err).Msg("Failed to record pending enrollment")
		return nil, nil, err
	}

	go func() {
		smsBody := strings.Builder{}
		smsBody.WriteString("Your verification code is: ")
		smsBody.WriteString(otpData.Otp)
		if err := p.SMSProvider.SendSMS(phone, smsBody.String()); err != nil {
			log.Debug().Msg("Failed to send sms")
		}
	}()

	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditMFAEnabledEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{Message: "Check your phone for the verification code"}, nil, nil
}

// generateAndStoreOTP mirrors the local `generateOTP` closures in login.go /
// resend_otp.go: generates a plaintext OTP, persists its HMAC digest via
// UpsertOTP (keyed by the user's email/phone), and returns the plaintext on
// the returned struct for the caller's email/SMS body. Not shared with those
// closures directly since they capture per-call locals (log, ctx); this is
// the package-level equivalent for the setup mutations.
func (p *provider) generateAndStoreOTP(ctx context.Context, user *schemas.User, expiresAt int64) (*schemas.OTP, error) {
	otp, err := utils.GenerateOTP()
	if err != nil {
		return nil, err
	}
	otpData, err := p.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
		Email:       refs.StringValue(user.Email),
		PhoneNumber: refs.StringValue(user.PhoneNumber),
		Otp:         crypto.HashOTP(otp, p.Config.JWTSecret),
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		return nil, err
	}
	otpData.Otp = otp
	return otpData, nil
}

// upsertUnverifiedAuthenticator creates the (user, method) Authenticator row
// if absent, or leaves an existing unverified one in place (a fresh OTP was
// just sent for it — the row's Secret field is unused for OTP methods, only
// VerifiedAt matters). Never touches an already-verified row: re-running
// setup after enrollment is a no-op enrollment-wise, only the send-a-code
// side effect repeats.
func (p *provider) upsertUnverifiedAuthenticator(ctx context.Context, userID, method string) error {
	existing, err := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, userID, method)
	if err == nil && existing != nil {
		return nil // already exists (verified or not) — nothing to create
	}
	_, err = p.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
		UserID: userID,
		Method: method,
	})
	return err
}

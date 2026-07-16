// internal/service/otp_mfa_setup.go
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
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// resolveOTPSetupCaller resolves the caller for EmailOTPMFASetup /
// SMSOTPMFASetup under either of two auth modes:
//
//  1. Bearer token / session (unchanged, existing behavior) — an
//     already-logged-in user adding a second factor from account settings.
//     Any email/phone_number param is ignored; the token already identifies
//     the user.
//  2. MFA session cookie — a caller in the token-withheld first-time-offer
//     state (mfaGateOfferAll) has no bearer token yet. Falls back to the
//     same cookie + email/phone_number identity-resolution pattern already
//     used by SkipMFASetup/LockMFA: resolve the user by the given
//     email/phone_number, then validate the MFA session cookie is actually
//     theirs.
//
// Returns Unauthenticated if neither mode resolves a caller.
func (p *provider) resolveOTPSetupCaller(ctx context.Context, meta RequestMetadata, params *model.OtpMfaSetupRequest) (*schemas.User, error) {
	if tokenData, err := p.callerTokenData(ctx, meta); err == nil && tokenData != nil && tokenData.UserID != "" {
		return p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	}

	gc := &gin.Context{Request: meta.Request}
	mfaSession, err := cookie.GetMfaSession(gc)
	if err != nil {
		return nil, Unauthenticated(`unauthorized`)
	}

	var email, phoneNumber string
	if params != nil {
		email = strings.TrimSpace(refs.StringValue(params.Email))
		phoneNumber = strings.TrimSpace(refs.StringValue(params.PhoneNumber))
	}
	if email == "" && phoneNumber == "" {
		// No identifier supplied (OAuth-return first-time-offer): resolve the
		// account from the session cookie alone.
		ownerID, _, oErr := p.MemoryStoreProvider.GetMfaSessionOwner(mfaSession)
		if oErr != nil {
			return nil, Unauthenticated(`unauthorized`)
		}
		user, uErr := p.StorageProvider.GetUserByID(ctx, ownerID)
		if user == nil || uErr != nil {
			return nil, Unauthenticated(`unauthorized`)
		}
		return user, nil
	}

	var user *schemas.User
	if email != "" {
		user, err = p.StorageProvider.GetUserByEmail(ctx, email)
	} else {
		user, err = p.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
	}
	if user == nil || err != nil {
		return nil, Unauthenticated(`unauthorized`)
	}

	if _, err := p.MemoryStoreProvider.GetMfaSession(user.ID, mfaSession); err != nil {
		return nil, Unauthenticated(`unauthorized`)
	}

	return user, nil
}

// EmailOTPMFASetup sends a one-time code to the caller's own email and
// creates (or refreshes) an unverified email-OTP Authenticator row.
// Permissions: authenticated caller (bearer token) — the settings-screen
// "add a second factor" action — OR, absent a token, the MFA session cookie
// plus params.email/phone_number for a caller in the withheld first-time-
// offer state. See resolveOTPSetupCaller.
func (p *provider) EmailOTPMFASetup(ctx context.Context, meta RequestMetadata, params *model.OtpMfaSetupRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "EmailOTPMFASetup").Logger()

	user, err := p.resolveOTPSetupCaller(ctx, meta, params)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve caller")
		return nil, nil, err
	}

	if !p.Config.EnableEmailOTP || !p.Config.IsEmailServiceEnabled {
		return nil, nil, FailedPrecondition("email OTP is not available on this server")
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
func (p *provider) SMSOTPMFASetup(ctx context.Context, meta RequestMetadata, params *model.OtpMfaSetupRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "SMSOTPMFASetup").Logger()

	user, err := p.resolveOTPSetupCaller(ctx, meta, params)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve caller")
		return nil, nil, err
	}

	if !p.Config.EnableSMSOTP || !p.Config.IsSMSServiceEnabled {
		return nil, nil, FailedPrecondition("SMS OTP is not available on this server")
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

// TOTPMFASetup generates a fresh TOTP secret/QR/recovery-codes for the
// caller to enroll as an MFA method. Unlike EmailOTPMFASetup/SMSOTPMFASetup,
// nothing is sent anywhere - generateTOTPEnrollment (login.go) already
// persists the unverified Authenticator row itself (via
// AuthenticatorProvider.Generate), so the enrollment payload is simply
// returned to the caller to scan/enter, then completed via
// VerifyOTP(is_totp: true) same as the login-gate TOTP flow.
// Permissions: same dual mode as EmailOTPMFASetup/SMSOTPMFASetup - see
// resolveOTPSetupCaller.
func (p *provider) TOTPMFASetup(ctx context.Context, meta RequestMetadata, params *model.OtpMfaSetupRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "TOTPMFASetup").Logger()

	user, err := p.resolveOTPSetupCaller(ctx, meta, params)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to resolve caller")
		return nil, nil, err
	}

	if !p.Config.EnableTOTPLogin {
		return nil, nil, FailedPrecondition("authenticator app MFA is not available on this server")
	}

	enrollment, err := p.generateTOTPEnrollment(ctx, user.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to generate totp enrollment")
		return nil, nil, err
	}

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

	return &model.AuthResponse{
		Message:                    `Proceed to totp verification screen`,
		ShouldShowTotpScreen:       refs.NewBoolRef(true),
		AuthenticatorScannerImage:  refs.NewStringRef(enrollment.ScannerImage),
		AuthenticatorSecret:        refs.NewStringRef(enrollment.Secret),
		AuthenticatorRecoveryCodes: enrollment.RecoveryCodes,
	}, nil, nil
}

// generateAndStoreOTP is the single OTP-generation implementation shared by
// login.go's email/SMS-verification and TOTP-alternative branches,
// resend_otp.go, and this file's Email/SMS-OTP-as-MFA setup: generates a
// plaintext OTP, persists its HMAC digest via UpsertOTP (keyed by the user's
// email/phone), and returns the plaintext on the returned struct for the
// caller's email/SMS body. Callers log their own error context at the call
// site rather than this method logging internally.
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

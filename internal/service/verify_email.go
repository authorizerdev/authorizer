package service

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// VerifyEmail verifies a user's email using a verification token, completing
// signup (or a magic-link login) and issuing a session. Transport-agnostic
// port of graphqlProvider.VerifyEmail.
//
// Permissions: none.
func (p *provider) VerifyEmail(ctx context.Context, meta RequestMetadata, params *model.VerifyEmailRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "VerifyEmail").Logger()
	side := &ResponseSideEffects{}

	verificationRequest, err := p.StorageProvider.GetVerificationRequestByToken(ctx, params.Token)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetVerificationRequestByToken")
		return nil, nil, InvalidArgument(`invalid verification token`)
	}

	// verify if token exists in db
	hostname := meta.HostURL
	claim, err := p.TokenProvider.ParseJWTToken(params.Token)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse jwt token")
		return nil, nil, InvalidArgument(`invalid verification token`)
	}

	if ok, err := p.TokenProvider.ValidateJWTClaims(claim, &token.AuthTokenConfig{
		HostName: hostname,
		Nonce:    verificationRequest.Nonce,
		User: &schemas.User{
			Email: &verificationRequest.Email,
		},
	}); !ok || err != nil {
		log.Debug().Err(err).Msg("Failed to validate jwt claims")
		return nil, nil, InvalidArgument(`invalid verification token`)
	}

	email := claim["sub"].(string)
	log.Debug().Str("email", email).Msg("Email verified successfully")
	user, err := p.StorageProvider.GetUserByEmail(ctx, email)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetUserByEmail")
		return nil, nil, err
	}

	if user.RevokedTimestamp != nil {
		log.Debug().Msg("User access has been revoked")
		return nil, nil, FailedPrecondition("user access has been revoked")
	}

	// A single check protecting every MFA branch below, mirroring login.go —
	// lockout is set only by explicit user action (lock_mfa), never inferred
	// here, and must block magic-link/email-verify completion exactly like
	// it blocks password login.
	if user.MFALockedAt != nil {
		log.Debug().Msg("User's MFA is locked, refusing login")
		return nil, nil, FailedPrecondition("your account's multi-factor authentication is locked; contact your administrator to regain access")
	}

	loginMethod := constants.AuthRecipeMethodBasicAuth
	if verificationRequest.Identifier == constants.VerificationTypeMagicLinkLogin {
		loginMethod = constants.AuthRecipeMethodMagicLinkLogin
	}

	isTOTPLoginEnabled := p.Config.EnableTOTPLogin
	isMFAEnabled := p.Config.EnableMFA

	// A verified Email-OTP second factor is challenged on enrollment alone,
	// mirroring login.go's identical early branch — ported here because this
	// endpoint used to fall straight into the TOTP/WebAuthn-only gate below
	// with no way to ever challenge an email/SMS-OTP factor at all.
	emailOTPAuthenticator, _ := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyEmailOTPAuthenticator)
	emailOTPEnrolled := emailOTPAuthenticator != nil && emailOTPAuthenticator.VerifiedAt != nil
	if effectiveMFAEnabled(p.Config, user) && isMFAEnabled && p.Config.EnableEmailOTP && p.Config.IsEmailServiceEnabled && emailOTPEnrolled {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
		otpData, err := p.generateAndStoreOTP(ctx, user, expiresAt)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to generate otp")
			return nil, nil, err
		}
		if err := p.setMFASession(meta, side, user.ID, expiresAt); err != nil {
			log.Debug().Err(err).Msg("Failed to set mfa session")
			return nil, nil, err
		}
		go func() {
			ctx := context.WithoutCancel(ctx)
			if err := p.EmailProvider.SendEmail([]string{refs.StringValue(user.Email)}, constants.VerificationTypeOTP, map[string]any{
				"user":         user.ToMap(),
				"organization": utils.GetOrganization(p.Config),
				"otp":          otpData.Otp,
			}); err != nil {
				log.Debug().Err(err).Msg("Failed to send otp email")
			}
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		}()
		return &model.AuthResponse{
			Message:                  "Please check email inbox for the OTP",
			ShouldShowEmailOtpScreen: refs.NewBoolRef(true),
		}, side, nil
	}
	// SMS-OTP twin of the email branch above.
	smsOTPAuthenticator, _ := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeySMSOTPAuthenticator)
	smsOTPEnrolled := smsOTPAuthenticator != nil && smsOTPAuthenticator.VerifiedAt != nil
	if effectiveMFAEnabled(p.Config, user) && isMFAEnabled && p.Config.EnableSMSOTP && p.Config.IsSMSServiceEnabled && smsOTPEnrolled {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
		otpData, err := p.generateAndStoreOTP(ctx, user, expiresAt)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to generate otp")
			return nil, nil, err
		}
		if err := p.setMFASession(meta, side, user.ID, expiresAt); err != nil {
			log.Debug().Err(err).Msg("Failed to set mfa session")
			return nil, nil, err
		}
		go func() {
			ctx := context.WithoutCancel(ctx)
			smsBody := strings.Builder{}
			smsBody.WriteString("Your verification code is: ")
			smsBody.WriteString(otpData.Otp)
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
			if err := p.SMSProvider.SendSMS(refs.StringValue(user.PhoneNumber), smsBody.String()); err != nil {
				log.Debug().Err(err).Msg("Failed to send sms")
			}
		}()
		return &model.AuthResponse{
			Message:                   "Please check text message for the OTP",
			ShouldShowMobileOtpScreen: refs.NewBoolRef(true),
		}, side, nil
	}

	// Gate runs whenever MFA applies at all, exactly like login.go/signup.go —
	// this used to be an ad-hoc TOTP-only check (refs.BoolValue(user.IsMultiFactorAuthEnabled)
	// && isMFAEnabled && isTOTPLoginEnabled) that silently skipped WebAuthn,
	// email/SMS-OTP-as-MFA, EnforceMFA, and HasSkippedMFASetupAt entirely — a
	// user whose only configured factor was WebAuthn or email/SMS-OTP (or
	// whose account required first-time MFA setup/enforcement) could
	// complete a magic-link login or signup-email-verification with zero MFA
	// challenge. Replaced with the same resolveMFAGate call every other
	// entry point uses. Reaching this point means neither email-OTP nor
	// SMS-OTP is the user's enrolled factor (those returned above), so
	// totpVerified/hasWebauthnCredential is the correct authenticatorVerified
	// set here — mirrors login.go's identical structure and reasoning.
	if isMFAEnabled {
		totpAuthenticator, totpErr := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
		totpVerified := totpErr == nil && totpAuthenticator != nil && totpAuthenticator.VerifiedAt != nil
		webauthnCreds, _ := p.StorageProvider.ListWebauthnCredentialsByUserID(ctx, user.ID)
		hasWebauthnCredential := len(webauthnCreds) > 0
		authenticatorVerified := totpVerified || hasWebauthnCredential
		gate := resolveMFAGate(
			effectiveMFAEnabled(p.Config, user),
			p.Config.EnforceMFA,
			authenticatorVerified,
			user.HasSkippedMFASetupAt != nil,
		)
		switch gate {
		case mfaGateBlockVerify:
			expiresAt := time.Now().Add(3 * time.Minute).Unix()
			if err := p.setMFASession(meta, side, user.ID, expiresAt); err != nil {
				log.Debug().Err(err).Msg("Failed to set mfa session")
				return nil, nil, err
			}
			res := &model.AuthResponse{Message: `Proceed to mfa verification`}
			if totpVerified && isTOTPLoginEnabled {
				res.ShouldShowTotpScreen = refs.NewBoolRef(true)
			}
			if hasWebauthnCredential {
				res.ShouldOfferWebauthnMfaVerify = refs.NewBoolRef(true)
			}
			return res, side, nil
		case mfaGateBlockEnroll, mfaGateOfferAll:
			expiresAt := time.Now().Add(3 * time.Minute).Unix()
			if err := p.setMFASession(meta, side, user.ID, expiresAt); err != nil {
				log.Debug().Err(err).Msg("Failed to set mfa session")
				return nil, nil, err
			}
			res := &model.AuthResponse{
				Message:                     `Proceed to mfa setup`,
				ShouldOfferWebauthnMfaSetup: refs.NewBoolRef(p.Config.EnableWebauthnMFA),
				ShouldOfferEmailOtpMfaSetup: refs.NewBoolRef(p.Config.EnableEmailOTP && p.Config.IsEmailServiceEnabled),
				ShouldOfferSmsOtpMfaSetup:   refs.NewBoolRef(p.Config.EnableSMSOTP && p.Config.IsSMSServiceEnabled),
			}
			if isTOTPLoginEnabled {
				enrollment, err := p.generateTOTPEnrollment(ctx, user.ID)
				if err != nil {
					log.Debug().Err(err).Msg("Failed to generate totp")
					return nil, nil, err
				}
				res.ShouldShowTotpScreen = refs.NewBoolRef(true)
				res.AuthenticatorScannerImage = refs.NewStringRef(enrollment.ScannerImage)
				res.AuthenticatorSecret = refs.NewStringRef(enrollment.Secret)
				res.AuthenticatorRecoveryCodes = enrollment.RecoveryCodes
			}
			return res, side, nil
		case mfaGateSkippedSetup, mfaGateNone:
			// fall through, nothing to do
		}
	}

	isSignUp := false
	if user.EmailVerifiedAt == nil {
		isSignUp = true
		// update email_verified_at in users table
		now := time.Now().Unix()
		user.EmailVerifiedAt = &now
		user, err = p.StorageProvider.UpdateUser(ctx, user)
		if err != nil {
			log.Debug().Err(err).Msg("failed UpdateUser")
			return nil, nil, err
		}
	}
	// delete from verification table
	err = p.StorageProvider.DeleteVerificationRequest(ctx, verificationRequest)
	if err != nil {
		log.Debug().Err(err).Msg("failed DeleteVerificationRequest")
		return nil, nil, err
	}

	roles := strings.Split(user.Roles, ",")
	scope := []string{"openid", "email", "profile"}
	code := ""
	// Not required as /oauth/token cannot be resumed from other tab
	// codeChallenge := ""
	nonce := ""
	if params.State != nil {
		// Get state from store
		authorizeState, _ := p.MemoryStoreProvider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				// Not required as /oauth/token cannot be resumed from other tab
				// codeChallenge = authorizeStateSplit[1]
			} else {
				nonce = authorizeState
			}
			go func() { _ = p.MemoryStoreProvider.RemoveState(refs.StringValue(params.State)) }()
		}
	}
	if nonce == "" {
		nonce = uuid.New().String()
	}
	// TokenProvider.CreateAuthToken takes *gin.Context but doesn't read from it.
	// Synthesize a minimal gin.Context wrapping the inbound *http.Request so the
	// call works for both gin and non-gin transports.
	gcShim := &gin.Context{Request: meta.Request}
	authToken, err := p.TokenProvider.CreateAuthToken(gcShim, &token.AuthTokenConfig{
		HostName:    hostname,
		User:        user,
		Roles:       roles,
		Scope:       scope,
		LoginMethod: loginMethod,
		Nonce:       nonce,
		Code:        code,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create auth token")
		return nil, nil, err
	}

	// Code challenge could be optional if PKCE flow is not used
	// Not required as /oauth/token cannot be resumed from other tab
	// if code != "" {
	// 	if err := memorystore.Provider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
	// 		log.Debug("SetState failed: ", err)
	// 		return nil, err
	// 	}
	// }
	go func() {
		ctx := context.WithoutCancel(ctx)
		if isSignUp {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, loginMethod, user)
			// User is also logged in with signup
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		} else {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		}

		if err := p.StorageProvider.AddSession(ctx, &schemas.Session{
			UserID:    user.ID,
			UserAgent: meta.UserAgent,
			IP:        meta.IPAddress,
		}); err != nil {
			log.Debug().Err(err).Msg("Failed to add session")
		}
	}()
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
	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res := &model.AuthResponse{
		Message:     `Email verified successfully.`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &expiresIn,
		User:        user.AsAPIUser(),
	}

	sessionKey := loginMethod + ":" + user.ID
	for _, c := range cookie.BuildSessionCookies(hostname, authToken.FingerPrintHash, p.Config.AppCookieSecure, cookie.ParseSameSite(p.Config.AppCookieSameSite)) {
		side.AddCookie(c)
	}
	_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}
	return res, side, nil
}

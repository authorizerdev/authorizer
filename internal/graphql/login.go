package graphql

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// genericLoginErr is the single error returned for any authentication failure.
// All distinct failure causes (user not found, wrong password, email not verified,
// wrong auth method, account revoked) collapse to this message to prevent user
// enumeration. The real reason is recorded in the debug log for ops visibility.
const genericLoginErrMsg = "invalid credentials"

// dummyBcryptHash is a precomputed bcrypt hash used to equalise the response
// time of the user-not-found path with the real password verification path.
// Without this, an attacker can distinguish "no such user" from "wrong
// password" by measuring response latency (no bcrypt vs one bcrypt).
var (
	dummyBcryptHash []byte
	dummyBcryptOnce sync.Once
)

// performDummyPasswordCheck runs a constant-cost bcrypt comparison whose result
// is intentionally discarded. Call it on the user-not-found / no-password
// branches so the request still does roughly the same amount of CPU work as a
// real authentication attempt.
func performDummyPasswordCheck(password string) {
	dummyBcryptOnce.Do(func() {
		// generated lazily so cost depends on bcrypt.DefaultCost at runtime
		dummyBcryptHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte(password))
}

// Login is the method to login a user.
// User can login with email or phone number, but not both
// Permissions: none
func (g *graphqlProvider) Login(ctx context.Context, params *model.LoginRequest) (*model.AuthResponse, error) {
	log := g.Log.With().Str("func", "Login").Logger()

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	isBasicAuthEnabled := g.Config.EnableBasicAuthentication
	isMobileBasicAuthEnabled := g.Config.EnableMobileBasicAuthentication
	email := refs.StringValue(params.Email)
	phoneNumber := refs.StringValue(params.PhoneNumber)
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, fmt.Errorf(`email or phone number is required`)
	}
	log = log.With().Str("email", email).Str("phone_number", phoneNumber).Logger()
	isEmailLogin := email != ""
	isMobileLogin := phoneNumber != ""
	if !isBasicAuthEnabled && isEmailLogin {
		log.Debug().Msg("Basic authentication is disabled")
		return nil, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	if !isMobileBasicAuthEnabled && isMobileLogin {
		log.Debug().Msg("Mobile basic authentication is disabled")
		return nil, fmt.Errorf(`mobile basic authentication is disabled for this instance`)
	}
	var user *schemas.User
	if isEmailLogin {
		user, err = g.StorageProvider.GetUserByEmail(ctx, email)
		if err == nil {
			log.Debug().Str("email", email).Msg("User found by email")
		}
	} else {
		user, err = g.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err == nil {
			log.Debug().Str("phone_number", phoneNumber).Msg("User found by phone number")
		}
	}
	if err != nil {
		log.Debug().Err(err).Str("reason", "user_not_found").Msg("login failed")
		metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusFailure)
		// Equalise response timing with the real bcrypt path so an attacker
		// cannot distinguish "no such user" from "wrong password" by latency.
		performDummyPasswordCheck(params.Password)
		return nil, fmt.Errorf(genericLoginErrMsg)
	}
	if user.RevokedTimestamp != nil {
		log.Debug().Str("reason", "account_revoked").Msg("login failed")
		metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusFailure)
		metrics.RecordSecurityEvent("account_revoked", "login_attempt")
		performDummyPasswordCheck(params.Password)
		return nil, fmt.Errorf(genericLoginErrMsg)
	}
	isEmailServiceEnabled := g.Config.IsEmailServiceEnabled
	isSMSServiceEnabled := g.Config.IsSMSServiceEnabled
	// If multi factor authentication is enabled and we need to generate OTP for mail / sms based MFA
	generateOTP := func(expiresAt int64) (*schemas.OTP, error) {
		otp, err := utils.GenerateOTP()
		if err != nil {
			log.Debug().Err(err).Msg("Failed to generate OTP")
			return nil, err
		}
		// Store the HMAC digest (defence-in-depth: an offline DB dump no
		// longer reveals usable codes). The plaintext is held in the
		// returned struct's Otp field for the caller's email/SMS body.
		otpData, err := g.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			Email:       refs.StringValue(user.Email),
			PhoneNumber: refs.StringValue(user.PhoneNumber),
			Otp:         crypto.HashOTP(otp, g.Config.JWTSecret),
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Debug().Msg("Failed to upsert otp")
			return nil, err
		}
		// Replace the persisted hash with the plaintext on the returned
		// struct so the caller can read otpData.Otp for email/SMS without
		// having to thread two values through the closure.
		otpData.Otp = otp
		return otpData, nil
	}
	setOTPMFaSession := func(expiresAt int64) error {
		mfaSession := uuid.NewString()
		err = g.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return err
		}
		cookie.SetMfaSession(gc, mfaSession, g.Config.AppCookieSecure)
		return nil
	}
	if isEmailLogin {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodBasicAuth) {
			log.Debug().Str("reason", "wrong_signup_method").Msg("login failed")
			performDummyPasswordCheck(params.Password)
			return nil, fmt.Errorf(genericLoginErrMsg)
		}

		if user.EmailVerifiedAt == nil {
			// Check if email service is enabled
			// Send email verification via otp
			if !isEmailServiceEnabled {
				log.Debug().Str("reason", "email_not_verified_no_email_service").Msg("login failed")
				performDummyPasswordCheck(params.Password)
				return nil, fmt.Errorf(genericLoginErrMsg)
			} else {
				if vreq, err := g.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup); err == nil && vreq != nil {
					// if verification request exists and not expired then return
					// if verification request exists and expired then delete it and proceed
					if vreq.ExpiresAt > time.Now().Unix() {
						log.Debug().Str("reason", "email_verification_pending").Msg("login failed")
						performDummyPasswordCheck(params.Password)
						return nil, fmt.Errorf(genericLoginErrMsg)
					} else {
						if err := g.StorageProvider.DeleteVerificationRequest(ctx, vreq); err != nil {
							log.Debug().Msg("Failed to delete verification request")
							return nil, err
						} else {
							log.Debug().Msg("Verification request deleted")
						}
					}
				}
				expiresAt := time.Now().Add(1 * time.Minute).Unix()
				otpData, err := generateOTP(expiresAt)
				if err != nil {
					log.Debug().Msg("Failed to generate otp")
					return nil, err
				}
				if err := setOTPMFaSession(expiresAt); err != nil {
					log.Debug().Msg("Failed to set mfa session")
					return nil, err
				}
				go func() {
					// exec it as go routine so that we can reduce the api latency
					if err := g.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]interface{}{
						"user":         user.ToMap(),
						"organization": utils.GetOrganization(g.Config),
						"otp":          otpData.Otp,
					}); err != nil {
						log.Debug().Msg("Failed to send otp email")
					}
					g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
				}()
				return &model.AuthResponse{
					Message:                  "Please check email inbox for the OTP",
					ShouldShowEmailOtpScreen: refs.NewBoolRef(isEmailLogin),
				}, nil
			}
		}
	} else {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodMobileBasicAuth) {
			log.Debug().Str("reason", "wrong_signup_method_phone").Msg("login failed")
			performDummyPasswordCheck(params.Password)
			return nil, fmt.Errorf(genericLoginErrMsg)
		}

		if user.PhoneNumberVerifiedAt == nil {
			if !isSMSServiceEnabled {
				log.Debug().Str("reason", "phone_not_verified_no_sms_service").Msg("login failed")
				performDummyPasswordCheck(params.Password)
				return nil, fmt.Errorf(genericLoginErrMsg)
			} else {
				expiresAt := time.Now().Add(1 * time.Minute).Unix()
				otpData, err := generateOTP(expiresAt)
				if err != nil {
					log.Debug().Msg("Failed to generate otp")
					return nil, err
				}
				if err := setOTPMFaSession(expiresAt); err != nil {
					log.Debug().Msg("Failed to set mfa session")
					return nil, err
				}
				go func() {
					smsBody := strings.Builder{}
					smsBody.WriteString("Your verification code is: ")
					smsBody.WriteString(otpData.Otp)
					g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
					if err := g.SMSProvider.SendSMS(phoneNumber, smsBody.String()); err != nil {
						log.Debug().Msg("Failed to send sms")
					}
				}()
				return &model.AuthResponse{
					Message:                   "Please check text message for the OTP",
					ShouldShowMobileOtpScreen: refs.NewBoolRef(isMobileLogin),
				}, nil
			}
		}
	}
	err = bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(params.Password))
	if err != nil {
		log.Debug().Str("reason", "bad_password").Msg("login failed")
		metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusFailure)
		metrics.RecordSecurityEvent("invalid_credentials", "bad_password")
		g.AuditProvider.LogEvent(audit.Event{
			Action:       constants.AuditLoginFailedEvent,
			ActorID:      user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   refs.StringValue(user.Email),
			ResourceType: constants.AuditResourceTypeUser,
			ResourceID:   user.ID,
			IPAddress:    utils.GetIP(gc.Request),
			UserAgent:    utils.GetUserAgent(gc.Request),
		})
		return nil, fmt.Errorf(genericLoginErrMsg)
	}
	roles := g.Config.DefaultRoles
	currentRoles := strings.Split(user.Roles, ",")
	if len(params.Roles) > 0 {
		if !validators.IsValidRoles(params.Roles, currentRoles) {
			log.Debug().Msg("Invalid roles")
			return nil, fmt.Errorf(`invalid roles`)
		}
		roles = params.Roles
	}
	scope := []string{"openid", "email", "profile"}
	if len(params.Scope) > 0 {
		scope = params.Scope
	}

	isMFAEnabled := g.Config.EnableMFA
	isTOTPLoginEnabled := g.Config.EnableTOTPLogin
	isMailOTPEnabled := g.Config.EnableEmailOTP
	isSMSOTPEnabled := g.Config.EnableSMSOTP

	// If multi factor authentication is enabled and is email based login and email otp is enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && isMFAEnabled && isMailOTPEnabled && isEmailServiceEnabled && isEmailLogin {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
		otpData, err := generateOTP(expiresAt)
		if err != nil {
			log.Debug().Msg("Failed to generate otp")
			return nil, err
		}
		if err := setOTPMFaSession(expiresAt); err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return nil, err
		}
		go func() {
			// exec it as go routine so that we can reduce the api latency
			if err := g.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]interface{}{
				"user":         user.ToMap(),
				"organization": utils.GetOrganization(g.Config),
				"otp":          otpData.Otp,
			}); err != nil {
				log.Debug().Msg("Failed to send otp email")
			}
			g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		}()
		return &model.AuthResponse{
			Message:                  "Please check email inbox for the OTP",
			ShouldShowEmailOtpScreen: refs.NewBoolRef(isMobileLogin),
		}, nil
	}
	// If multi factor authentication is enabled and is sms based login and sms otp is enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && isMFAEnabled && isSMSOTPEnabled && isSMSServiceEnabled && isMobileLogin {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
		otpData, err := generateOTP(expiresAt)
		if err != nil {
			log.Debug().Msg("Failed to generate otp")
			return nil, err
		}
		if err := setOTPMFaSession(expiresAt); err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return nil, err
		}
		go func() {
			smsBody := strings.Builder{}
			smsBody.WriteString("Your verification code is: ")
			smsBody.WriteString(otpData.Otp)
			g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			if err := g.SMSProvider.SendSMS(phoneNumber, smsBody.String()); err != nil {
				log.Debug().Msg("Failed to send sms")
			}
		}()
		return &model.AuthResponse{
			Message:                   "Please check text message for the OTP",
			ShouldShowMobileOtpScreen: refs.NewBoolRef(isMobileLogin),
		}, nil
	}
	// If mfa enabled and also totp enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && isMFAEnabled && isTOTPLoginEnabled {
		expiresAt := time.Now().Add(3 * time.Minute).Unix()
		if err := setOTPMFaSession(expiresAt); err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return nil, err
		}
		authenticator, err := g.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
		if err != nil || authenticator == nil || authenticator.VerifiedAt == nil {
			// generate totp
			// Generate a base64 URL and initiate the registration for TOTP
			authConfig, err := g.AuthenticatorProvider.Generate(ctx, user.ID)
			if err != nil {
				log.Debug().Msg("Failed to generate totp")
				return nil, err
			}
			recoveryCodes := []*string{}
			for _, code := range authConfig.RecoveryCodes {
				recoveryCodes = append(recoveryCodes, refs.NewStringRef(code))
			}
			// when user is first time registering for totp
			res := &model.AuthResponse{
				Message:                    `Proceed to totp verification screen`,
				ShouldShowTotpScreen:       refs.NewBoolRef(true),
				AuthenticatorScannerImage:  refs.NewStringRef(authConfig.ScannerImage),
				AuthenticatorSecret:        refs.NewStringRef(authConfig.Secret),
				AuthenticatorRecoveryCodes: recoveryCodes,
			}
			return res, nil
		} else {
			//when user is already register for totp
			res := &model.AuthResponse{
				Message:              `Proceed to totp screen`,
				ShouldShowTotpScreen: refs.NewBoolRef(true),
			}
			return res, nil
		}
	}

	code := ""
	codeChallenge := ""
	nonce := ""
	oidcNonce := ""
	authorizeRedirectURI := ""
	if params.State != nil {
		// Get state from store
		authorizeState, _ := g.MemoryStoreProvider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				codeChallenge = authorizeStateSplit[1]
				if len(authorizeStateSplit) > 2 {
					oidcNonce = authorizeStateSplit[2]
				}
				// RFC 6749 §4.1.3: redirect_uri from /authorize for validation at /oauth/token
				if len(authorizeStateSplit) > 3 {
					authorizeRedirectURI = authorizeStateSplit[3]
				}
			} else {
				nonce = authorizeState
			}
			g.MemoryStoreProvider.RemoveState(refs.StringValue(params.State))
		}
	}

	if nonce == "" {
		nonce = uuid.New().String()
	}
	hostname := parsers.GetHost(gc)
	authToken, err := g.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
		User:        user,
		Roles:       roles,
		Scope:       scope,
		Nonce:       nonce,
		OIDCNonce:   oidcNonce,
		Code:        code,
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
		HostName:    hostname,
	})
	if err != nil {
		log.Debug().Msg("Failed to create auth token")
		return nil, err
	}

	// Code challenge could be optional if PKCE flow is not used
	if code != "" {
		if err := g.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash+"@@"+oidcNonce+"@@"+authorizeRedirectURI); err != nil {
			log.Debug().Msg("Failed to set state")
			return nil, err
		}
	}

	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res := &model.AuthResponse{
		Message:     `Logged in successfully`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &expiresIn,
		User:        user.AsAPIUser(),
	}

	cookie.SetSession(gc, authToken.FingerPrintHash, g.Config.AppCookieSecure)
	sessionStoreKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	g.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	g.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		g.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	go func() {
		// Register event
		if isEmailLogin {
			g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		} else {
			g.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		}
		// Record session
		g.StorageProvider.AddSession(ctx, &schemas.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		})
	}()
	metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusSuccess)
	metrics.ActiveSessions.Inc()
	g.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditLoginSuccessEvent,
		ActorID:      user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeSession,
		ResourceID:   user.ID,
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
	})

	return res, nil
}

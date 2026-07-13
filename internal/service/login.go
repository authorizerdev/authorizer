package service

import (
	"strings"
	"sync"
	"time"

	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

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

// loginGenericErrMsg is the single error returned for any authentication
// failure. All distinct failure causes (user not found, wrong password, email
// not verified, wrong auth method, account revoked) collapse to this message to
// prevent user enumeration. The real reason is recorded in the debug log for
// ops visibility.
const loginGenericErrMsg = "invalid credentials"

// loginDummyBcryptHash is a precomputed bcrypt hash used to equalise the
// response time of the user-not-found path with the real password verification
// path. Without this, an attacker can distinguish "no such user" from "wrong
// password" by measuring response latency (no bcrypt vs one bcrypt).
var (
	loginDummyBcryptHash []byte
	loginDummyBcryptOnce sync.Once
)

// loginPerformDummyPasswordCheck runs a constant-cost bcrypt comparison whose
// result is intentionally discarded. Call it on the user-not-found / no-password
// branches so the request still does roughly the same amount of CPU work as a
// real authentication attempt.
func loginPerformDummyPasswordCheck(password string) {
	loginDummyBcryptOnce.Do(func() {
		// generated lazily so cost depends on bcrypt.DefaultCost at runtime
		loginDummyBcryptHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	})
	_ = bcrypt.CompareHashAndPassword(loginDummyBcryptHash, []byte(password))
}

// totpEnrollment is a freshly generated (unverified) TOTP enrollment
// payload, shared by both the mfaGateBlockEnroll (forced) and
// mfaGateOfferSetup (optional) paths of the TOTP MFA branch below.
type totpEnrollment struct {
	ScannerImage  string
	Secret        string
	RecoveryCodes []*string
}

// generateTOTPEnrollment generates a new TOTP secret/QR/recovery-codes for
// userID. Extracted so the TOTP MFA branch doesn't duplicate this call across
// its "block until enrolled" and "offer setup" cases.
func (p *provider) generateTOTPEnrollment(ctx context.Context, userID string) (*totpEnrollment, error) {
	authConfig, err := p.AuthenticatorProvider.Generate(ctx, userID)
	if err != nil {
		return nil, err
	}
	recoveryCodes := []*string{}
	for _, code := range authConfig.RecoveryCodes {
		recoveryCodes = append(recoveryCodes, refs.NewStringRef(code))
	}
	return &totpEnrollment{
		ScannerImage:  authConfig.ScannerImage,
		Secret:        authConfig.Secret,
		RecoveryCodes: recoveryCodes,
	}, nil
}

// Login authenticates a user with email or phone number (not both).
// Transport-agnostic port of graphqlProvider.Login.
//
// Permissions: none.
func (p *provider) Login(ctx context.Context, meta RequestMetadata, params *model.LoginRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Login").Logger()
	side := &ResponseSideEffects{}

	isBasicAuthEnabled := p.Config.EnableBasicAuthentication
	isMobileBasicAuthEnabled := p.Config.EnableMobileBasicAuthentication
	email := refs.StringValue(params.Email)
	phoneNumber := refs.StringValue(params.PhoneNumber)
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, nil, InvalidArgument(`email or phone number is required`)
	}
	log = log.With().Str("email", email).Str("phone_number", phoneNumber).Logger()
	isEmailLogin := email != ""
	isMobileLogin := phoneNumber != ""
	if !isBasicAuthEnabled && isEmailLogin {
		log.Debug().Msg("Basic authentication is disabled")
		return nil, nil, FailedPrecondition(`basic authentication is disabled for this instance`)
	}
	if !isMobileBasicAuthEnabled && isMobileLogin {
		log.Debug().Msg("Mobile basic authentication is disabled")
		return nil, nil, FailedPrecondition(`mobile basic authentication is disabled for this instance`)
	}
	var user *schemas.User
	var err error
	if isEmailLogin {
		user, err = p.StorageProvider.GetUserByEmail(ctx, email)
		if err == nil {
			log.Debug().Str("email", email).Msg("User found by email")
		}
	} else {
		user, err = p.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		if err == nil {
			log.Debug().Str("phone_number", phoneNumber).Msg("User found by phone number")
		}
	}
	if err != nil {
		log.Debug().Err(err).Str("reason", "user_not_found").Msg("login failed")
		metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusFailure)
		// Equalise response timing with the real bcrypt path so an attacker
		// cannot distinguish "no such user" from "wrong password" by latency.
		loginPerformDummyPasswordCheck(params.Password)
		return nil, nil, Unauthenticated(loginGenericErrMsg)
	}
	if user.RevokedTimestamp != nil {
		log.Debug().Str("reason", "account_revoked").Msg("login failed")
		metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusFailure)
		metrics.RecordSecurityEvent("account_revoked", "login_attempt")
		loginPerformDummyPasswordCheck(params.Password)
		return nil, nil, Unauthenticated(loginGenericErrMsg)
	}
	isEmailServiceEnabled := p.Config.IsEmailServiceEnabled
	isSMSServiceEnabled := p.Config.IsSMSServiceEnabled
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
		otpData, err := p.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			Email:       refs.StringValue(user.Email),
			PhoneNumber: refs.StringValue(user.PhoneNumber),
			Otp:         crypto.HashOTP(otp, p.Config.JWTSecret),
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
		err = p.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return err
		}
		for _, c := range cookie.BuildMfaSessionCookies(meta.HostURL, mfaSession, p.Config.AppCookieSecure) {
			side.AddCookie(c)
		}
		return nil
	}
	if isEmailLogin {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodBasicAuth) {
			log.Debug().Str("reason", "wrong_signup_method").Msg("login failed")
			loginPerformDummyPasswordCheck(params.Password)
			return nil, nil, Unauthenticated(loginGenericErrMsg)
		}

		if user.EmailVerifiedAt == nil {
			// Check if email service is enabled
			// Send email verification via otp
			if !isEmailServiceEnabled {
				log.Debug().Str("reason", "email_not_verified_no_email_service").Msg("login failed")
				loginPerformDummyPasswordCheck(params.Password)
				return nil, nil, Unauthenticated(loginGenericErrMsg)
			} else {
				if vreq, err := p.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup); err == nil && vreq != nil {
					// if verification request exists and not expired then return
					// if verification request exists and expired then delete it and proceed
					if vreq.ExpiresAt > time.Now().Unix() {
						log.Debug().Str("reason", "email_verification_pending").Msg("login failed")
						loginPerformDummyPasswordCheck(params.Password)
						return nil, nil, Unauthenticated(loginGenericErrMsg)
					} else {
						if err := p.StorageProvider.DeleteVerificationRequest(ctx, vreq); err != nil {
							log.Debug().Msg("Failed to delete verification request")
							return nil, nil, err
						} else {
							log.Debug().Msg("Verification request deleted")
						}
					}
				}
				expiresAt := time.Now().Add(1 * time.Minute).Unix()
				otpData, err := generateOTP(expiresAt)
				if err != nil {
					log.Debug().Msg("Failed to generate otp")
					return nil, nil, err
				}
				if err := setOTPMFaSession(expiresAt); err != nil {
					log.Debug().Msg("Failed to set mfa session")
					return nil, nil, err
				}
				go func() {
					ctx := context.WithoutCancel(ctx)
					// exec it as go routine so that we can reduce the api latency
					if err := p.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]any{
						"user":         user.ToMap(),
						"organization": utils.GetOrganization(p.Config),
						"otp":          otpData.Otp,
					}); err != nil {
						log.Debug().Msg("Failed to send otp email")
					}
					_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
				}()
				return &model.AuthResponse{
					Message:                  "Please check email inbox for the OTP",
					ShouldShowEmailOtpScreen: refs.NewBoolRef(isEmailLogin),
				}, side, nil
			}
		}
	} else {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodMobileBasicAuth) {
			log.Debug().Str("reason", "wrong_signup_method_phone").Msg("login failed")
			loginPerformDummyPasswordCheck(params.Password)
			return nil, nil, Unauthenticated(loginGenericErrMsg)
		}

		if user.PhoneNumberVerifiedAt == nil {
			if !isSMSServiceEnabled {
				log.Debug().Str("reason", "phone_not_verified_no_sms_service").Msg("login failed")
				loginPerformDummyPasswordCheck(params.Password)
				return nil, nil, Unauthenticated(loginGenericErrMsg)
			} else {
				expiresAt := time.Now().Add(1 * time.Minute).Unix()
				otpData, err := generateOTP(expiresAt)
				if err != nil {
					log.Debug().Msg("Failed to generate otp")
					return nil, nil, err
				}
				if err := setOTPMFaSession(expiresAt); err != nil {
					log.Debug().Msg("Failed to set mfa session")
					return nil, nil, err
				}
				go func() {
					ctx := context.WithoutCancel(ctx)
					smsBody := strings.Builder{}
					smsBody.WriteString("Your verification code is: ")
					smsBody.WriteString(otpData.Otp)
					_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
					if err := p.SMSProvider.SendSMS(phoneNumber, smsBody.String()); err != nil {
						log.Debug().Msg("Failed to send sms")
					}
				}()
				return &model.AuthResponse{
					Message:                   "Please check text message for the OTP",
					ShouldShowMobileOtpScreen: refs.NewBoolRef(isMobileLogin),
				}, side, nil
			}
		}
	}
	if user.Password == nil {
		// A basic_auth user with no stored hash (e.g. a pre-fix Couchbase
		// record that never persisted one) must fail the same way as a
		// wrong password, not nil-pointer-dereference.
		loginPerformDummyPasswordCheck(params.Password)
		err = bcrypt.ErrMismatchedHashAndPassword
	} else {
		err = bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(params.Password))
	}
	if err != nil {
		log.Debug().Str("reason", "bad_password").Msg("login failed")
		metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusFailure)
		metrics.RecordSecurityEvent("invalid_credentials", "bad_password")
		p.AuditProvider.LogEvent(audit.Event{
			Action:   constants.AuditLoginFailedEvent,
			Protocol: meta.Protocol, ActorID: user.ID,
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   refs.StringValue(user.Email),
			ResourceType: constants.AuditResourceTypeUser,
			ResourceID:   user.ID,
			IPAddress:    meta.IPAddress,
			UserAgent:    meta.UserAgent,
		})
		return nil, nil, Unauthenticated(loginGenericErrMsg)
	}
	roles := p.Config.DefaultRoles
	currentRoles := strings.Split(user.Roles, ",")
	if len(params.Roles) > 0 {
		if !validators.IsValidRoles(params.Roles, currentRoles) {
			log.Debug().Msg("Invalid roles")
			return nil, nil, InvalidArgument(`invalid roles`)
		}
		roles = params.Roles
	}
	scope := []string{"openid", "email", "profile"}
	if len(params.Scope) > 0 {
		scope = params.Scope
	}

	isMFAEnabled := p.Config.EnableMFA
	isTOTPLoginEnabled := p.Config.EnableTOTPLogin
	isMailOTPEnabled := p.Config.EnableEmailOTP
	isSMSOTPEnabled := p.Config.EnableSMSOTP

	// If multi factor authentication is enabled and is email based login and email otp is enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && isMFAEnabled && isMailOTPEnabled && isEmailServiceEnabled && isEmailLogin {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
		otpData, err := generateOTP(expiresAt)
		if err != nil {
			log.Debug().Msg("Failed to generate otp")
			return nil, nil, err
		}
		if err := setOTPMFaSession(expiresAt); err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return nil, nil, err
		}
		go func() {
			ctx := context.WithoutCancel(ctx)
			// exec it as go routine so that we can reduce the api latency
			if err := p.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]any{
				"user":         user.ToMap(),
				"organization": utils.GetOrganization(p.Config),
				"otp":          otpData.Otp,
			}); err != nil {
				log.Debug().Msg("Failed to send otp email")
			}
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		}()
		return &model.AuthResponse{
			Message:                  "Please check email inbox for the OTP",
			ShouldShowEmailOtpScreen: refs.NewBoolRef(isEmailLogin),
		}, side, nil
	}
	// If multi factor authentication is enabled and is sms based login and sms otp is enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && isMFAEnabled && isSMSOTPEnabled && isSMSServiceEnabled && isMobileLogin {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
		otpData, err := generateOTP(expiresAt)
		if err != nil {
			log.Debug().Msg("Failed to generate otp")
			return nil, nil, err
		}
		if err := setOTPMFaSession(expiresAt); err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return nil, nil, err
		}
		go func() {
			ctx := context.WithoutCancel(ctx)
			smsBody := strings.Builder{}
			smsBody.WriteString("Your verification code is: ")
			smsBody.WriteString(otpData.Otp)
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			if err := p.SMSProvider.SendSMS(phoneNumber, smsBody.String()); err != nil {
				log.Debug().Msg("Failed to send sms")
			}
		}()
		return &model.AuthResponse{
			Message:                   "Please check text message for the OTP",
			ShouldShowMobileOtpScreen: refs.NewBoolRef(isMobileLogin),
		}, side, nil
	}
	// If mfa enabled and also totp enabled
	if isMFAEnabled && isTOTPLoginEnabled {
		authenticator, authErr := p.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
		authenticatorVerified := authErr == nil && authenticator != nil && authenticator.VerifiedAt != nil
		gate := resolveMFAGate(
			refs.BoolValue(user.IsMultiFactorAuthEnabled),
			p.Config.EnforceMFA,
			authenticatorVerified,
			user.HasSkippedMFASetupAt != nil,
		)
		switch gate {
		case mfaGateBlockVerify:
			expiresAt := time.Now().Add(3 * time.Minute).Unix()
			if err := setOTPMFaSession(expiresAt); err != nil {
				log.Debug().Msg("Failed to set mfa session")
				return nil, nil, err
			}
			return &model.AuthResponse{
				Message:              `Proceed to totp screen`,
				ShouldShowTotpScreen: refs.NewBoolRef(true),
			}, side, nil
		case mfaGateBlockEnroll:
			expiresAt := time.Now().Add(3 * time.Minute).Unix()
			if err := setOTPMFaSession(expiresAt); err != nil {
				log.Debug().Msg("Failed to set mfa session")
				return nil, nil, err
			}
			enrollment, err := p.generateTOTPEnrollment(ctx, user.ID)
			if err != nil {
				log.Debug().Msg("Failed to generate totp")
				return nil, nil, err
			}
			return &model.AuthResponse{
				Message:                    `Proceed to totp verification screen`,
				ShouldShowTotpScreen:       refs.NewBoolRef(true),
				AuthenticatorScannerImage:  refs.NewStringRef(enrollment.ScannerImage),
				AuthenticatorSecret:        refs.NewStringRef(enrollment.Secret),
				AuthenticatorRecoveryCodes: enrollment.RecoveryCodes,
			}, side, nil
		case mfaGateOfferSetup:
			enrollment, err := p.generateTOTPEnrollment(ctx, user.ID)
			if err != nil {
				log.Debug().Msg("Failed to generate totp for optional setup")
				return nil, nil, err
			}
			// Falls through to normal token issuance below, with the offer
			// flag and enrollment payload attached after CreateAuthToken.
			side.PendingTOTPOffer = enrollment
		case mfaGateSkippedSetup:
			side.OfferMFASetupQuiet = true
		case mfaGateNone:
			// fall through, nothing to do
		}
	}

	code := ""
	codeChallenge := ""
	nonce := ""
	oidcNonce := ""
	authorizeRedirectURI := ""
	if params.State != nil {
		// Get state from store
		authorizeState, _ := p.MemoryStoreProvider.GetState(refs.StringValue(params.State))
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
			_ = p.MemoryStoreProvider.RemoveState(refs.StringValue(params.State))
		}
	}

	if nonce == "" {
		nonce = uuid.New().String()
	}
	hostname := meta.HostURL
	// TokenProvider.CreateAuthToken takes *gin.Context but doesn't read from
	// it (only AccessToken-getter and ID-token-getter helpers in the same
	// file do). Synthesize a minimal gin.Context wrapping the inbound
	// *http.Request so the call works for both gin and non-gin transports.
	// TODO(grpc): refactor TokenProvider to take *http.Request directly.
	gcShim := &gin.Context{Request: meta.Request}
	authToken, err := p.TokenProvider.CreateAuthToken(gcShim, &token.AuthTokenConfig{
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
		return nil, nil, err
	}

	// Code challenge could be optional if PKCE flow is not used
	if code != "" {
		if err := p.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash+"@@"+oidcNonce+"@@"+authorizeRedirectURI); err != nil {
			log.Debug().Msg("Failed to set state")
			return nil, nil, err
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
	if side.PendingTOTPOffer != nil {
		res.ShouldOfferMfaSetup = refs.NewBoolRef(true)
		res.AuthenticatorScannerImage = refs.NewStringRef(side.PendingTOTPOffer.ScannerImage)
		res.AuthenticatorSecret = refs.NewStringRef(side.PendingTOTPOffer.Secret)
		res.AuthenticatorRecoveryCodes = side.PendingTOTPOffer.RecoveryCodes
	}

	for _, c := range cookie.BuildSessionCookies(meta.HostURL, authToken.FingerPrintHash, p.Config.AppCookieSecure, cookie.ParseSameSite(p.Config.AppCookieSameSite)) {
		side.AddCookie(c)
	}
	sessionStoreKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	_ = p.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	_ = p.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		_ = p.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	go func() {
		ctx := context.WithoutCancel(ctx)
		// Register event
		if isEmailLogin {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		} else {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		}
		// Record session
		_ = p.StorageProvider.AddSession(ctx, &schemas.Session{
			UserID:    user.ID,
			UserAgent: meta.UserAgent,
			IP:        meta.IPAddress,
		})
	}()
	metrics.RecordAuthEvent(metrics.EventLogin, metrics.StatusSuccess)
	metrics.ActiveSessions.Inc()
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditLoginSuccessEvent,
		Protocol: meta.Protocol, ActorID: user.ID,
		ActorType:    constants.AuditActorTypeUser,
		ActorEmail:   refs.StringValue(user.Email),
		ResourceType: constants.AuditResourceTypeSession,
		ResourceID:   user.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return res, side, nil
}

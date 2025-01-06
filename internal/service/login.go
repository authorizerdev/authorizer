package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// Login is the method to login a user.
// User can login with email or phone number, but not both
// Permissions: none
func (s *service) Login(ctx context.Context, params *model.LoginInput) (*model.AuthResponse, error) {
	log := s.Log.With().Str("func", "Login").Logger()

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	isBasicAuthDisabled := s.Config.DisableBasicAuthentication
	isMobileBasicAuthDisabled := s.Config.DisableMobileBasicAuthentication
	email := refs.StringValue(params.Email)
	phoneNumber := refs.StringValue(params.PhoneNumber)
	if email == "" && phoneNumber == "" {
		log.Debug().Msg("Email or phone number is required")
		return nil, fmt.Errorf(`email or phone number is required`)
	}
	log = log.With().Str("email", email).Str("phone_number", phoneNumber).Logger()
	isEmailLogin := email != ""
	isMobileLogin := phoneNumber != ""
	if isBasicAuthDisabled && isEmailLogin {
		log.Debug().Msg("Basic authentication is disabled")
		return nil, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	if isMobileBasicAuthDisabled && isMobileLogin {
		log.Debug().Msg("Mobile basic authentication is disabled")
		return nil, fmt.Errorf(`mobile basic authentication is disabled for this instance`)
	}
	var user *schemas.User
	if isEmailLogin {
		user, err = s.StorageProvider.GetUserByEmail(ctx, email)
		log.Debug().Str("email", email).Msg("User found by email")
	} else {
		user, err = s.StorageProvider.GetUserByPhoneNumber(ctx, phoneNumber)
		log.Debug().Str("phone_number", phoneNumber).Msg("User found by phone number")
	}
	if err != nil {
		return nil, fmt.Errorf(`user not found`)
	}
	if user.RevokedTimestamp != nil {
		log.Debug().Msg("User access has been revoked")
		return nil, fmt.Errorf(`user access has been revoked`)
	}
	isEmailServiceEnabled := s.Config.IsEmailServiceEnabled
	isSMSServiceEnabled := s.Config.IsSMSServiceEnabled
	// If multi factor authentication is enabled and we need to generate OTP for mail / sms based MFA
	generateOTP := func(expiresAt int64) (*schemas.OTP, error) {
		otp := utils.GenerateOTP()
		otpData, err := s.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
			Email:       refs.StringValue(user.Email),
			PhoneNumber: refs.StringValue(user.PhoneNumber),
			Otp:         otp,
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Debug().Msg("Failed to upsert otp")
			return nil, err
		}
		return otpData, nil
	}
	setOTPMFaSession := func(expiresAt int64) error {
		mfaSession := uuid.NewString()
		err = s.MemoryStoreProvider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return err
		}
		cookie.SetMfaSession(gc, mfaSession)
		return nil
	}
	if isEmailLogin {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodBasicAuth) {
			log.Debug().Msg("User signup method is not email basic auth")
			return nil, fmt.Errorf(`user has not signed up email & password`)
		}

		if user.EmailVerifiedAt == nil {
			// Check if email service is enabled
			// Send email verification via otp
			if !isEmailServiceEnabled {
				log.Debug().Msg("Email service is not enabled")
				return nil, fmt.Errorf(`email not verified`)
			} else {
				if vreq, err := s.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup); err == nil && vreq != nil {
					// if verification request exists and not expired then return
					// if verification request exists and expired then delete it and proceed
					if vreq.ExpiresAt <= time.Now().Unix() {
						if err := s.StorageProvider.DeleteVerificationRequest(ctx, vreq); err != nil {
							log.Debug().Msg("Failed to delete verification request")
							// continue with the flow
						}
					} else {
						log.Debug().Msg("Email verification pending")
						return nil, fmt.Errorf(`email verification pending`)
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
					if err := s.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]interface{}{
						"user":         user.ToMap(),
						"organization": utils.GetOrganization(s.Config),
						"otp":          otpData.Otp,
					}); err != nil {
						log.Debug().Msg("Failed to send otp email")
					}
					s.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
				}()
				return &model.AuthResponse{
					Message:                  "Please check email inbox for the OTP",
					ShouldShowEmailOtpScreen: refs.NewBoolRef(isEmailLogin),
				}, nil
			}
		}
	} else {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodMobileBasicAuth) {
			log.Debug().Msg("User signup method is not phone number basic auth")
			return nil, fmt.Errorf(`user has not signed up with phone number & password`)
		}

		if user.PhoneNumberVerifiedAt == nil {
			if !isSMSServiceEnabled {
				log.Debug().Msg("SMS service is not enabled")
				return nil, fmt.Errorf(`phone number is not verified and sms service is not enabled`)
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
					s.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
					if err := s.SMSProvider.SendSMS(phoneNumber, smsBody.String()); err != nil {
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
		log.Debug().Msg("Bad user credentials")
		return nil, fmt.Errorf(`bad user credentials`)
	}
	roles := strings.Split(s.Config.DefaultRoles, ",")
	currentRoles := strings.Split(user.Roles, ",")
	if len(params.Roles) > 0 {
		if !validators.IsValidRoles(params.Roles, currentRoles) {
			log.Debug().Msg("Invalid roles")
			return nil, fmt.Errorf(`invalid roles`)
		}
		roles = params.Roles
	}
	scope := []string{"openid", "email", "profile"}
	if params.Scope != nil && len(scope) > 0 {
		scope = params.Scope
	}

	isMFADisabled := s.Config.DisableMFA
	isTOTPLoginDisabled := s.DisableTOTPLogin
	isMailOTPDisabled := s.DisableEmailOTP
	isSMSOTPDisabled := s.DisableSMSOTP

	// If multi factor authentication is enabled and is email based login and email otp is enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && !isMFADisabled && !isMailOTPDisabled && isEmailServiceEnabled && isEmailLogin {
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
			if err := s.EmailProvider.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]interface{}{
				"user":         user.ToMap(),
				"organization": utils.GetOrganization(s.Config),
				"otp":          otpData.Otp,
			}); err != nil {
				log.Debug().Msg("Failed to send otp email")
			}
			s.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		}()
		return &model.AuthResponse{
			Message:                  "Please check email inbox for the OTP",
			ShouldShowEmailOtpScreen: refs.NewBoolRef(isMobileLogin),
		}, nil
	}
	// If multi factor authentication is enabled and is sms based login and sms otp is enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && !isMFADisabled && !isSMSOTPDisabled && isSMSServiceEnabled && isMobileLogin {
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
			s.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			if err := s.SMSProvider.SendSMS(phoneNumber, smsBody.String()); err != nil {
				log.Debug().Msg("Failed to send sms")
			}
		}()
		return &model.AuthResponse{
			Message:                   "Please check text message for the OTP",
			ShouldShowMobileOtpScreen: refs.NewBoolRef(isMobileLogin),
		}, nil
	}
	// If mfa enabled and also totp enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && !isMFADisabled && !isTOTPLoginDisabled {
		expiresAt := time.Now().Add(3 * time.Minute).Unix()
		if err := setOTPMFaSession(expiresAt); err != nil {
			log.Debug().Msg("Failed to set mfa session")
			return nil, err
		}
		authenticator, err := s.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
		if err != nil || authenticator == nil || authenticator.VerifiedAt == nil {
			// generate totp
			// Generate a base64 URL and initiate the registration for TOTP
			authConfig, err := s.AuthenticatorProvider.Generate(ctx, user.ID)
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
	if params.State != nil {
		// Get state from store
		authorizeState, _ := s.MemoryStoreProvider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				codeChallenge = authorizeStateSplit[1]
			} else {
				nonce = authorizeState
			}
			go s.MemoryStoreProvider.RemoveState(refs.StringValue(params.State))
		}
	}

	if nonce == "" {
		nonce = uuid.New().String()
	}
	// gc, user, roles, scope, constants.AuthRecipeMethodBasicAuth, nonce, code
	authToken, err := s.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
		User:        user,
		Roles:       roles,
		Scope:       scope,
		Nonce:       nonce,
		Code:        code,
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
	})
	if err != nil {
		log.Debug().Msg("Failed to create auth token")
		return nil, err
	}

	// TODO add to other login options as well
	// Code challenge could be optional if PKCE flow is not used
	if code != "" {
		if err := s.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
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

	cookie.SetSession(gc, authToken.FingerPrintHash)
	sessionStoreKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	s.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	s.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		s.MemoryStoreProvider.SetUserSession(sessionStoreKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	go func() {
		// Register event
		if isEmailLogin {
			s.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		} else {
			s.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		}
		// Record session
		s.StorageProvider.AddSession(ctx, &schemas.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		})
	}()

	return res, nil
}

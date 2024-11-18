package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/data_store/db"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	mailService "github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/smsproviders"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// LoginResolver is a resolver for login mutation
// User can login with email or phone number, but not both
func LoginResolver(ctx context.Context, params model.LoginInput) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication)
	if err != nil {
		log.Debug("Error getting basic auth disabled: ", err)
		isBasicAuthDisabled = true
	}

	isMobileBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication)
	if err != nil {
		log.Debug("Error getting mobile basic auth disabled: ", err)
		isMobileBasicAuthDisabled = true
	}

	email := refs.StringValue(params.Email)
	phoneNumber := refs.StringValue(params.PhoneNumber)
	if email == "" && phoneNumber == "" {
		log.Debug("Email or phone number is required")
		return res, fmt.Errorf(`email or phone number is required`)
	}
	log := log.WithFields(log.Fields{
		"email":        refs.StringValue(params.Email),
		"phone_number": refs.StringValue(params.PhoneNumber),
	})
	isEmailLogin := email != ""
	isMobileLogin := phoneNumber != ""
	if isBasicAuthDisabled {
		log.Debug("Basic authentication is disabled.")
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	if isMobileBasicAuthDisabled && isMobileLogin {
		log.Debug("Mobile basic authentication is disabled.")
		return res, fmt.Errorf(`mobile basic authentication is disabled for this instance`)
	}
	var user *models.User
	if isEmailLogin {
		user, err = db.Provider.GetUserByEmail(ctx, email)
	} else {
		user, err = db.Provider.GetUserByPhoneNumber(ctx, phoneNumber)
	}
	if err != nil {
		log.Debug("Failed to get user: ", err)
		return res, fmt.Errorf(`user not found`)
	}
	if user.RevokedTimestamp != nil {
		log.Debug("User access is revoked")
		return res, fmt.Errorf(`user access has been revoked`)
	}
	isEmailServiceEnabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsEmailServiceEnabled)
	if err != nil || !isEmailServiceEnabled {
		log.Debug("Email service not enabled: ", err)
	}
	isSMSServiceEnabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsSMSServiceEnabled)
	if err != nil || !isSMSServiceEnabled {
		log.Debug("SMS service not enabled: ", err)
	}
	// If multi factor authentication is enabled and we need to generate OTP for mail / sms based MFA
	generateOTP := func(expiresAt int64) (*models.OTP, error) {
		otp := utils.GenerateOTP()
		otpData, err := db.Provider.UpsertOTP(ctx, &models.OTP{
			Email:       refs.StringValue(user.Email),
			PhoneNumber: refs.StringValue(user.PhoneNumber),
			Otp:         otp,
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Debug("Failed to add otp: ", err)
			return nil, err
		}
		return otpData, nil
	}
	setOTPMFaSession := func(expiresAt int64) error {
		mfaSession := uuid.NewString()
		err = memorystore.Provider.SetMfaSession(user.ID, mfaSession, expiresAt)
		if err != nil {
			log.Debug("Failed to add mfasession: ", err)
			return err
		}
		cookie.SetMfaSession(gc, mfaSession)
		return nil
	}
	if isEmailLogin {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodBasicAuth) {
			log.Debug("User signup method is not basic auth")
			return res, fmt.Errorf(`user has not signed up email & password`)
		}

		if user.EmailVerifiedAt == nil {
			// Check if email service is enabled
			// Send email verification via otp
			if !isEmailServiceEnabled {
				log.Debug("User email is not verified and email service is not enabled")
				return res, fmt.Errorf(`email not verified`)
			} else {
				if vreq, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup); err == nil && vreq != nil {
					// if verification request exists and not expired then return
					// if verification request exists and expired then delete it and proceed
					if vreq.ExpiresAt <= time.Now().Unix() {
						if err := db.Provider.DeleteVerificationRequest(ctx, vreq); err != nil {
							log.Debug("Failed to delete verification request: ", err)
							// continue with the flow
						}
					} else {
						log.Debug("Verification request exists. Please verify email")
						return res, fmt.Errorf(`email verification pending`)
					}
				}
				expiresAt := time.Now().Add(1 * time.Minute).Unix()
				otpData, err := generateOTP(expiresAt)
				if err != nil {
					log.Debug("Failed to generate otp: ", err)
					return nil, err
				}
				if err := setOTPMFaSession(expiresAt); err != nil {
					log.Debug("Failed to set mfa session: ", err)
					return nil, err
				}
				go func() {
					// exec it as go routine so that we can reduce the api latency
					if err := mailService.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]interface{}{
						"user":         user.ToMap(),
						"organization": utils.GetOrganization(),
						"otp":          otpData.Otp,
					}); err != nil {
						log.Debug("Failed to send otp email: ", err)
					}
					utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
				}()
				return &model.AuthResponse{
					Message:                  "Please check email inbox for the OTP",
					ShouldShowEmailOtpScreen: refs.NewBoolRef(isEmailLogin),
				}, nil
			}
		}
	} else {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodMobileBasicAuth) {
			log.Debug("User signup method is not mobile basic auth")
			return res, fmt.Errorf(`user has not signed up with phone number & password`)
		}

		if user.PhoneNumberVerifiedAt == nil {
			if !isSMSServiceEnabled {
				log.Debug("User phone number is not verified")
				return res, fmt.Errorf(`phone number is not verified and sms service is not enabled`)
			} else {
				expiresAt := time.Now().Add(1 * time.Minute).Unix()
				otpData, err := generateOTP(expiresAt)
				if err != nil {
					log.Debug("Failed to generate otp: ", err)
					return nil, err
				}
				if err := setOTPMFaSession(expiresAt); err != nil {
					log.Debug("Failed to set mfa session: ", err)
					return nil, err
				}
				go func() {
					smsBody := strings.Builder{}
					smsBody.WriteString("Your verification code is: ")
					smsBody.WriteString(otpData.Otp)
					utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
					if err := smsproviders.SendSMS(phoneNumber, smsBody.String()); err != nil {
						log.Debug("Failed to send sms: ", err)
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
		log.Debug("Failed to compare password: ", err)
		return res, fmt.Errorf(`bad user credentials`)
	}
	defaultRolesString, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDefaultRoles)
	roles := []string{}
	if err != nil {
		log.Debug("Error getting default roles: ", err)
		defaultRolesString = ""
	} else {
		roles = strings.Split(defaultRolesString, ",")
	}
	currentRoles := strings.Split(user.Roles, ",")
	if len(params.Roles) > 0 {
		if !validators.IsValidRoles(params.Roles, currentRoles) {
			log.Debug("Invalid roles: ", params.Roles)
			return res, fmt.Errorf(`invalid roles`)
		}
		roles = params.Roles
	}
	scope := []string{"openid", "email", "profile"}
	if params.Scope != nil && len(scope) > 0 {
		scope = params.Scope
	}

	isMFADisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMultiFactorAuthentication)
	if err != nil || !isMFADisabled {
		log.Debug("MFA service not enabled: ", err)
	}

	isTOTPLoginDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableTOTPLogin)
	if err != nil || !isTOTPLoginDisabled {
		log.Debug("totp service not enabled: ", err)
	}

	isMailOTPDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMailOTPLogin)
	if err != nil || !isMailOTPDisabled {
		log.Debug("mail OTP service not enabled: ", err)
	}

	isSMSOTPDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisablePhoneVerification)
	if err != nil || !isSMSOTPDisabled {
		log.Debug("sms OTP service not enabled: ", err)
	}

	// If multi factor authentication is enabled and is email based login and email otp is enabled
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && !isMFADisabled && !isMailOTPDisabled && isEmailServiceEnabled && isEmailLogin {
		expiresAt := time.Now().Add(1 * time.Minute).Unix()
		otpData, err := generateOTP(expiresAt)
		if err != nil {
			log.Debug("Failed to generate otp: ", err)
			return nil, err
		}
		if err := setOTPMFaSession(expiresAt); err != nil {
			log.Debug("Failed to set mfa session: ", err)
			return nil, err
		}
		go func() {
			// exec it as go routine so that we can reduce the api latency
			if err := mailService.SendEmail([]string{email}, constants.VerificationTypeOTP, map[string]interface{}{
				"user":         user.ToMap(),
				"organization": utils.GetOrganization(),
				"otp":          otpData.Otp,
			}); err != nil {
				log.Debug("Failed to send otp email: ", err)
			}
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
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
			log.Debug("Failed to generate otp: ", err)
			return nil, err
		}
		if err := setOTPMFaSession(expiresAt); err != nil {
			log.Debug("Failed to set mfa session: ", err)
			return nil, err
		}
		go func() {
			smsBody := strings.Builder{}
			smsBody.WriteString("Your verification code is: ")
			smsBody.WriteString(otpData.Otp)
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			if err := smsproviders.SendSMS(phoneNumber, smsBody.String()); err != nil {
				log.Debug("Failed to send sms: ", err)
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
			log.Debug("Failed to set mfa session: ", err)
			return nil, err
		}
		authenticator, err := db.Provider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
		if err != nil || authenticator == nil || authenticator.VerifiedAt == nil {
			// generate totp
			// Generate a base64 URL and initiate the registration for TOTP
			authConfig, err := authenticators.Provider.Generate(ctx, user.ID)
			if err != nil {
				log.Debug("error while generating base64 url: ", err)
				return nil, err
			}
			recoveryCodes := []*string{}
			for _, code := range authConfig.RecoveryCodes {
				recoveryCodes = append(recoveryCodes, refs.NewStringRef(code))
			}
			// when user is first time registering for totp
			res = &model.AuthResponse{
				Message:                    `Proceed to totp verification screen`,
				ShouldShowTotpScreen:       refs.NewBoolRef(true),
				AuthenticatorScannerImage:  refs.NewStringRef(authConfig.ScannerImage),
				AuthenticatorSecret:        refs.NewStringRef(authConfig.Secret),
				AuthenticatorRecoveryCodes: recoveryCodes,
			}
			return res, nil
		} else {
			//when user is already register for totp
			res = &model.AuthResponse{
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
		authorizeState, _ := memorystore.Provider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				codeChallenge = authorizeStateSplit[1]
			} else {
				nonce = authorizeState
			}
			go memorystore.Provider.RemoveState(refs.StringValue(params.State))
		}
	}

	if nonce == "" {
		nonce = uuid.New().String()
	}
	authToken, err := token.CreateAuthToken(gc, user, roles, scope, constants.AuthRecipeMethodBasicAuth, nonce, code)
	if err != nil {
		log.Debug("Failed to create auth token", err)
		return res, err
	}

	// TODO add to other login options as well
	// Code challenge could be optional if PKCE flow is not used
	if code != "" {
		if err := memorystore.Provider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash); err != nil {
			log.Debug("SetState failed: ", err)
			return res, err
		}
	}

	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res = &model.AuthResponse{
		Message:     `Logged in successfully`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &expiresIn,
		User:        user.AsAPIUser(),
	}

	cookie.SetSession(gc, authToken.FingerPrintHash)
	sessionStoreKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	go func() {
		// Register event
		if isEmailLogin {
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		} else {
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		}
		// Record session
		db.Provider.AddSession(ctx, &models.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		})
	}()

	return res, nil
}

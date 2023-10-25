package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	mailService "github.com/authorizerdev/authorizer/server/email"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/smsproviders"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
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
		return res, fmt.Errorf(`bad user credentials`)
	}
	if user.RevokedTimestamp != nil {
		log.Debug("User access is revoked")
		return res, fmt.Errorf(`user access has been revoked`)
	}
	if isEmailLogin {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodBasicAuth) {
			log.Debug("User signup method is not basic auth")
			return res, fmt.Errorf(`user has not signed up email & password`)
		}

		if user.EmailVerifiedAt == nil {
			log.Debug("User email is not verified")
			return res, fmt.Errorf(`email not verified`)
		}
	} else {
		if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodMobileBasicAuth) {
			log.Debug("User signup method is not mobile basic auth")
			return res, fmt.Errorf(`user has not signed up with phone number & password`)
		}

		if user.PhoneNumberVerifiedAt == nil {
			log.Debug("User phone number is not verified")
			return res, fmt.Errorf(`phone number is not verified`)
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
	isEmailServiceEnabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsEmailServiceEnabled)
	if err != nil || !isEmailServiceEnabled {
		log.Debug("Email service not enabled: ", err)
	}
	isSMSServiceEnabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsSMSServiceEnabled)
	if err != nil || !isSMSServiceEnabled {
		log.Debug("SMS service not enabled: ", err)
	}

	isMFADisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMultiFactorAuthentication)
	if err != nil || !isMFADisabled {
		log.Debug("MFA service not enabled: ", err)
	}

	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && !isMFADisabled {
		otp := utils.GenerateOTP()
		expires := time.Now().Add(1 * time.Minute).Unix()
		otpData, err := db.Provider.UpsertOTP(ctx, &models.OTP{
			Email:     refs.StringValue(user.Email),
			Otp:       otp,
			ExpiresAt: expires,
		})
		if err != nil {
			log.Debug("Failed to add otp: ", err)
			return nil, err
		}

		mfaSession := uuid.NewString()
		err = memorystore.Provider.SetMfaSession(user.ID, mfaSession, expires)
		if err != nil {
			log.Debug("Failed to add mfasession: ", err)
			return nil, err
		}
		cookie.SetMfaSession(gc, mfaSession)
		if isEmailServiceEnabled && isEmailLogin {
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
		} else if isSMSServiceEnabled && isMobileLogin {
			smsBody := strings.Builder{}
			smsBody.WriteString("Your verification code is: ")
			smsBody.WriteString(otpData.Otp)
			go func() {
				utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
				if err := smsproviders.SendSMS(phoneNumber, smsBody.String()); err != nil {
					log.Debug("Failed to send sms: ", err)
				}
			}()
		}
		return &model.AuthResponse{
			Message:                   "Please check the OTP in",
			ShouldShowEmailOtpScreen:  refs.NewBoolRef(isEmailLogin),
			ShouldShowMobileOtpScreen: refs.NewBoolRef(isMobileLogin),
		}, nil
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

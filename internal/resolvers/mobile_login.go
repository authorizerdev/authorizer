package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/smsproviders"
	"github.com/authorizerdev/authorizer/internal/storage/db"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// MobileLoginResolver is a resolver for mobile login mutation
func MobileLoginResolver(ctx context.Context, params model.MobileLoginInput) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	isBasicAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication)
	if err != nil {
		log.Debug("Error getting mobile basic auth disabled: ", err)
		isBasicAuthDisabled = true
	}

	if isBasicAuthDisabled {
		log.Debug("Basic authentication is disabled.")
		return res, fmt.Errorf(`phone number based basic authentication is disabled for this instance`)
	}

	log := log.WithFields(log.Fields{
		"phone_number": params.PhoneNumber,
	})

	user, err := db.Provider.GetUserByPhoneNumber(ctx, params.PhoneNumber)
	if err != nil {
		log.Debug("Failed to get user by phone number: ", err)
		return res, fmt.Errorf(`bad user credentials`)
	}

	if user.RevokedTimestamp != nil {
		log.Debug("User access is revoked")
		return res, fmt.Errorf(`user access has been revoked`)
	}

	if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodMobileBasicAuth) {
		log.Debug("User signup method is not mobile basic auth")
		return res, fmt.Errorf(`user has not signed up with phone number & password`)
	}

	if user.PhoneNumberVerifiedAt == nil {
		log.Debug("User phone number is not verified")
		return res, fmt.Errorf(`phone number is not verified`)
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

	disablePhoneVerification, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisablePhoneVerification)
	if err != nil {
		log.Debug("Error getting disable phone verification: ", err)
	}
	if disablePhoneVerification {
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
	}
	isSMSServiceEnabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyIsSMSServiceEnabled)
	if err != nil || !isSMSServiceEnabled {
		log.Debug("SMS service not enabled: ", err)
	}
	if disablePhoneVerification {
		now := time.Now().Unix()
		user.PhoneNumberVerifiedAt = &now
	}
	isMFADisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMultiFactorAuthentication)
	if err != nil || !isMFADisabled {
		log.Debug("MFA service not enabled: ", err)
	}
	if !disablePhoneVerification && isSMSServiceEnabled && !isMFADisabled {
		duration, _ := time.ParseDuration("10m")
		smsCode := utils.GenerateOTP()

		smsBody := strings.Builder{}
		smsBody.WriteString("Your verification code is: ")
		smsBody.WriteString(smsCode)
		expires := time.Now().Add(duration).Unix()
		_, err := db.Provider.UpsertOTP(ctx, &models.OTP{
			PhoneNumber: params.PhoneNumber,
			Otp:         smsCode,
			ExpiresAt:   expires,
		})
		if err != nil {
			log.Debug("error while upserting OTP: ", err.Error())
			return nil, err
		}

		mfaSession := uuid.NewString()
		err = memorystore.Provider.SetMfaSession(user.ID, mfaSession, expires)
		if err != nil {
			log.Debug("Failed to add mfasession: ", err)
			return nil, err
		}
		cookie.SetMfaSession(gc, mfaSession)

		go func() {
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
			smsproviders.SendSMS(params.PhoneNumber, smsBody.String())
		}()
		return &model.AuthResponse{
			Message:                   "Please check the OTP",
			ShouldShowMobileOtpScreen: refs.NewBoolRef(true),
		}, nil
	}

	scope := []string{"openid", "email", "profile"}
	if params.Scope != nil && len(scope) > 0 {
		scope = params.Scope
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

	authToken, err := token.CreateAuthToken(gc, user, roles, scope, constants.AuthRecipeMethodMobileBasicAuth, nonce, code)
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
	sessionStoreKey := constants.AuthRecipeMethodMobileBasicAuth + ":" + user.ID
	memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	go func() {
		utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodMobileBasicAuth, user)
		db.Provider.AddSession(ctx, &models.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		})
	}()

	return res, nil
}

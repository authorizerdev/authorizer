package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/email"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
)

// LoginResolver is a resolver for login mutation
func LoginResolver(ctx context.Context, params model.LoginInput) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	isBasiAuthDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication)
	if err != nil {
		log.Debug("Error getting basic auth disabled: ", err)
		isBasiAuthDisabled = true
	}

	if isBasiAuthDisabled {
		log.Debug("Basic authentication is disabled.")
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}

	log := log.WithFields(log.Fields{
		"email": params.Email,
	})
	params.Email = strings.ToLower(params.Email)
	user, err := db.Provider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		log.Debug("Failed to get user by email: ", err)
		return res, fmt.Errorf(`bad user credentials`)
	}

	if user.RevokedTimestamp != nil {
		log.Debug("User access is revoked")
		return res, fmt.Errorf(`user access has been revoked`)
	}

	if !strings.Contains(user.SignupMethods, constants.AuthRecipeMethodBasicAuth) {
		log.Debug("User signup method is not basic auth")
		return res, fmt.Errorf(`user has not signed up email & password`)
	}

	if user.EmailVerifiedAt == nil {
		log.Debug("User email is not verified")
		return res, fmt.Errorf(`email not verified`)
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

	isMFADisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMultiFactorAuthentication)
	if err != nil || !isEmailServiceEnabled {
		log.Debug("MFA service not enabled: ", err)
	}

	// If email service is not enabled continue the process in any way
	if refs.BoolValue(user.IsMultiFactorAuthEnabled) && isEmailServiceEnabled && !isMFADisabled {
		otp := utils.GenerateOTP()
		otpData, err := db.Provider.UpsertOTP(ctx, &models.OTP{
			Email:     user.Email,
			Otp:       otp,
			ExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
		})
		if err != nil {
			log.Debug("Failed to add otp: ", err)
			return nil, err
		}

		go func() {
			// exec it as go routine so that we can reduce the api latency
			go email.SendEmail([]string{params.Email}, constants.VerificationTypeOTP, map[string]interface{}{
				"user":         user.ToMap(),
				"organization": utils.GetOrganization(),
				"otp":          otpData.Otp,
			})
			if err != nil {
				log.Debug("Failed to send otp email: ", err)
			}
		}()

		return &model.AuthResponse{
			Message:             "Please check the OTP in your inbox",
			ShouldShowOtpScreen: refs.NewBoolRef(true),
		}, nil
	}

	authToken, err := token.CreateAuthToken(gc, user, roles, scope, constants.AuthRecipeMethodBasicAuth)
	if err != nil {
		log.Debug("Failed to create auth token", err)
		return res, err
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
	memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash)
	memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		memorystore.Provider.SetUserSession(sessionStoreKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token)
	}

	go func() {
		utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, constants.AuthRecipeMethodBasicAuth, user)
		db.Provider.AddSession(ctx, models.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		})
	}()

	return res, nil
}

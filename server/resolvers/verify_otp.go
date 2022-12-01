package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	log "github.com/sirupsen/logrus"
)

// VerifyOtpResolver resolver for verify otp mutation
func VerifyOtpResolver(ctx context.Context, params model.VerifyOTPRequest) (*model.AuthResponse, error) {
	var res *model.AuthResponse
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	otp, err := db.Provider.GetOTPByEmail(ctx, params.Email)
	if err != nil {
		log.Debug("Failed to get otp request by email: ", err)
		return res, fmt.Errorf(`invalid email: %s`, err.Error())
	}
	if params.Otp != otp.Otp {
		log.Debug("Failed to verify otp request: Incorrect value")
		return res, fmt.Errorf(`invalid otp`)
	}

	expiresIn := otp.ExpiresAt - time.Now().Unix()

	if expiresIn < 0 {
		log.Debug("Failed to verify otp request: Timeout")
		return res, fmt.Errorf("otp expired")
	}

	user, err := db.Provider.GetUserByEmail(ctx, params.Email)
	if err != nil {
		log.Debug("Failed to get user by email: ", err)
		return res, err
	}

	isSignUp := user.EmailVerifiedAt == nil

	// TODO - Add Login method in DB when we introduce OTP for social media login
	loginMethod := constants.AuthRecipeMethodBasicAuth

	roles := strings.Split(user.Roles, ",")
	scope := []string{"openid", "email", "profile"}
	authToken, err := token.CreateAuthToken(gc, user, roles, scope, loginMethod)
	if err != nil {
		log.Debug("Failed to create auth token: ", err)
		return res, err
	}

	go func() {
		db.Provider.DeleteOTP(gc, otp)
		if isSignUp {
			utils.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, loginMethod, user)
		} else {
			utils.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		}

		db.Provider.AddSession(ctx, models.Session{
			UserID:    user.ID,
			UserAgent: utils.GetUserAgent(gc.Request),
			IP:        utils.GetIP(gc.Request),
		})
	}()

	authTokenExpiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if authTokenExpiresIn <= 0 {
		authTokenExpiresIn = 1
	}

	res = &model.AuthResponse{
		Message:     `OTP verified successfully.`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &authTokenExpiresIn,
		User:        user.AsAPIUser(),
	}

	sessionKey := loginMethod + ":" + user.ID
	cookie.SetSession(gc, authToken.FingerPrintHash)
	memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash)
	memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		memorystore.Provider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token)
	}
	return res, nil
}

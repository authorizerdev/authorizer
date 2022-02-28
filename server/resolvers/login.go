package resolvers

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"golang.org/x/crypto/bcrypt"
)

// LoginResolver is a resolver for login mutation
func LoginResolver(ctx context.Context, params model.LoginInput) (*model.AuthResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.AuthResponse
	if err != nil {
		return res, err
	}

	if envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication) {
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}

	params.Email = strings.ToLower(params.Email)
	user, err := db.Provider.GetUserByEmail(params.Email)
	if err != nil {
		return res, fmt.Errorf(`user with this email not found`)
	}

	if !strings.Contains(user.SignupMethods, constants.SignupMethodBasicAuth) {
		return res, fmt.Errorf(`user has not signed up email & password`)
	}

	if user.EmailVerifiedAt == nil {
		return res, fmt.Errorf(`email not verified`)
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(params.Password))

	if err != nil {
		log.Println("compare password error:", err)
		return res, fmt.Errorf(`invalid password`)
	}
	roles := envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyDefaultRoles)
	currentRoles := strings.Split(user.Roles, ",")
	if len(params.Roles) > 0 {
		if !utils.IsValidRoles(currentRoles, params.Roles) {
			return res, fmt.Errorf(`invalid roles`)
		}

		roles = params.Roles
	}

	authToken, err := token.CreateAuthToken(user, roles)
	if err != nil {
		return res, err
	}
	sessionstore.SetUserSession(user.ID, authToken.FingerPrint, authToken.RefreshToken.Token)
	cookie.SetCookie(gc, authToken.AccessToken.Token, authToken.RefreshToken.Token, authToken.FingerPrintHash)
	utils.SaveSessionInDB(user.ID, gc)

	res = &model.AuthResponse{
		Message:     `Logged in successfully`,
		AccessToken: &authToken.AccessToken.Token,
		ExpiresAt:   &authToken.AccessToken.ExpiresAt,
		User:        user.AsAPIUser(),
	}

	return res, nil
}

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

	if user.RevokedTimestamp != nil {
		return res, fmt.Errorf(`user access has been revoked`)
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

	scope := []string{"openid", "email", "profile"}
	if params.Scope != nil && len(scope) > 0 {
		scope = params.Scope
	}

	authToken, err := token.CreateAuthToken(gc, user, roles, scope)
	if err != nil {
		return res, err
	}

	expiresIn := int64(1800)
	res = &model.AuthResponse{
		Message:     `Logged in successfully`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &expiresIn,
		User:        user.AsAPIUser(),
	}

	cookie.SetSession(gc, authToken.FingerPrintHash)
	sessionstore.SetState(authToken.FingerPrintHash, authToken.FingerPrint+"@"+user.ID)
	sessionstore.SetState(authToken.AccessToken.Token, authToken.FingerPrint+"@"+user.ID)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		sessionstore.SetState(authToken.RefreshToken.Token, authToken.FingerPrint+"@"+user.ID)
	}

	go utils.SaveSessionInDB(gc, user.ID)

	return res, nil
}

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
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
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
	user, err := db.Provider.GetUserByEmail(params.Email)
	if err != nil {
		log.Debug("Failed to get user by email: ", err)
		return res, fmt.Errorf(`user with this email not found`)
	}

	if user.RevokedTimestamp != nil {
		log.Debug("User access is revoked")
		return res, fmt.Errorf(`user access has been revoked`)
	}

	if !strings.Contains(user.SignupMethods, constants.SignupMethodBasicAuth) {
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
		return res, fmt.Errorf(`invalid password`)
	}

	roles, err := memorystore.Provider.GetSliceStoreEnvVariable(constants.EnvKeyDefaultRoles)
	if err != nil {
		log.Debug("Error getting default roles: ", err)
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

	authToken, err := token.CreateAuthToken(gc, user, roles, scope)
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
	memorystore.Provider.SetState(authToken.FingerPrintHash, authToken.FingerPrint+"@"+user.ID)
	memorystore.Provider.SetState(authToken.AccessToken.Token, authToken.FingerPrint+"@"+user.ID)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		memorystore.Provider.SetState(authToken.RefreshToken.Token, authToken.FingerPrint+"@"+user.ID)
	}

	go db.Provider.AddSession(models.Session{
		UserID:    user.ID,
		UserAgent: utils.GetUserAgent(gc.Request),
		IP:        utils.GetIP(gc.Request),
	})

	return res, nil
}

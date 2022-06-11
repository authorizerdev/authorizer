package resolvers

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// SessionResolver is a resolver for session query
// TODO allow validating with code and code verifier instead of cookie (PKCE flow)
func SessionResolver(ctx context.Context, params *model.SessionQueryInput) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	sessionToken, err := cookie.GetSession(gc)
	if err != nil {
		log.Debug("Failed to get session token", err)
		return res, errors.New("unauthorized")
	}

	// get session from cookie
	claims, err := token.ValidateBrowserSession(gc, sessionToken)
	if err != nil {
		log.Debug("Failed to validate session token", err)
		return res, errors.New("unauthorized")
	}
	userID := claims.Subject

	log := log.WithFields(log.Fields{
		"user_id": userID,
	})

	user, err := db.Provider.GetUserByID(userID)
	if err != nil {
		return res, err
	}

	// refresh token has "roles" as claim
	claimRoleInterface := claims.Roles
	claimRoles := []string{}
	claimRoles = append(claimRoles, claimRoleInterface...)

	if params != nil && params.Roles != nil && len(params.Roles) > 0 {
		for _, v := range params.Roles {
			if !utils.StringSliceContains(claimRoles, v) {
				log.Debug("User does not have required role: ", claimRoles, v)
				return res, fmt.Errorf(`unauthorized`)
			}
		}
	}

	scope := []string{"openid", "email", "profile"}
	if params != nil && params.Scope != nil && len(scope) > 0 {
		scope = params.Scope
	}

	authToken, err := token.CreateAuthToken(gc, user, claimRoles, scope)
	if err != nil {
		log.Debug("Failed to create auth token: ", err)
		return res, err
	}

	// rollover the session for security
	memorystore.Provider.RemoveState(sessionToken)
	memorystore.Provider.SetUserSession(user.ID, authToken.FingerPrintHash, authToken.FingerPrint)
	memorystore.Provider.SetUserSession(user.ID, authToken.AccessToken.Token, authToken.FingerPrint)
	cookie.SetSession(gc, authToken.FingerPrintHash)

	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res = &model.AuthResponse{
		Message:     `Session token refreshed`,
		AccessToken: &authToken.AccessToken.Token,
		ExpiresIn:   &expiresIn,
		IDToken:     &authToken.IDToken.Token,
		User:        user.AsAPIUser(),
	}

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		memorystore.Provider.SetUserSession(user.ID, authToken.RefreshToken.Token, authToken.FingerPrint)
	}

	return res, nil
}

package resolvers

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// LogoutResolver is a resolver for logout mutation
func LogoutResolver(ctx context.Context) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	tokenData, err := token.GetUserIDFromSessionOrAccessToken(gc)
	if err != nil {
		log.Debug("Failed GetUserIDFromSessionOrAccessToken: ", err)
		return nil, err
	}

	sessionKey := tokenData.UserID
	if tokenData.LoginMethod != "" {
		sessionKey = tokenData.LoginMethod + ":" + tokenData.UserID
	}

	memorystore.Provider.DeleteUserSession(sessionKey, tokenData.Nonce)
	cookie.DeleteSession(gc)

	res := &model.Response{
		Message: "Logged out successfully",
	}

	return res, nil
}

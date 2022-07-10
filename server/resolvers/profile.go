package resolvers

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// ProfileResolver is a resolver for profile query
func ProfileResolver(ctx context.Context) (*model.User, error) {
	var res *model.User

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	accessToken, err := token.GetAccessToken(gc)
	if err != nil {
		log.Debug("Failed to get access token: ", err)
		return res, err
	}

	claims, err := token.ValidateAccessToken(gc, accessToken)
	if err != nil {
		log.Debug("Failed to validate access token: ", err)
		return res, err
	}

	userID := claims["sub"].(string)

	log := log.WithFields(log.Fields{
		"user_id": userID,
	})
	user, err := db.Provider.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug("Failed to get user: ", err)
		return res, err
	}

	return user.AsAPIUser(), nil
}

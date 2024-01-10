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
	tokenData, err := token.GetUserIDFromSessionOrAccessToken(gc)
	if err != nil {
		log.Debug("Failed GetUserIDFromSessionOrAccessToken: ", err)
		return res, err
	}
	log := log.WithFields(log.Fields{
		"user_id": tokenData.UserID,
	})
	user, err := db.Provider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug("Failed to get user: ", err)
		return res, err
	}

	return user.AsAPIUser(), nil
}

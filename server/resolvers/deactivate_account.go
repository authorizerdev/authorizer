package resolvers

import (
	"context"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	log "github.com/sirupsen/logrus"
)

// DeactivateAccountResolver is the resolver for the deactivate_account field.
func DeactivateAccountResolver(ctx context.Context) (*model.Response, error) {
	var res *model.Response
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
		log.Debug("Failed to get user by id: ", err)
		return res, err
	}
	now := time.Now().Unix()
	user.RevokedTimestamp = &now
	user, err = db.Provider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug("Failed to update user: ", err)
		return res, err
	}
	go func() {
		memorystore.Provider.DeleteAllUserSessions(user.ID)
		utils.RegisterEvent(ctx, constants.UserDeactivatedWebhookEvent, "", user)
	}()
	res = &model.Response{
		Message: `user account deactivated successfully`,
	}
	return res, nil
}

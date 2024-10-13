package resolvers

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// RevokeAccessResolver is a resolver for revoking user access
func RevokeAccessResolver(ctx context.Context, params model.UpdateAccessInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return res, fmt.Errorf("unauthorized")
	}

	log := log.WithFields(log.Fields{
		"user_id": params.UserID,
	})
	user, err := db.Provider.GetUserByID(ctx, params.UserID)
	if err != nil {
		log.Debug("Failed to get user by ID: ", err)
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
		utils.RegisterEvent(ctx, constants.UserAccessRevokedWebhookEvent, "", user)
	}()

	res = &model.Response{
		Message: `user access revoked successfully`,
	}

	return res, nil
}

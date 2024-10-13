package resolvers

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// EnableAccessResolver is a resolver for enabling user access
func EnableAccessResolver(ctx context.Context, params model.UpdateAccessInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin.")
		return res, fmt.Errorf("unauthorized")
	}

	log := log.WithFields(log.Fields{
		"user_id": params.UserID,
	})

	user, err := db.Provider.GetUserByID(ctx, params.UserID)
	if err != nil {
		log.Debug("Failed to get user from DB: ", err)
		return res, err
	}

	user.RevokedTimestamp = nil

	user, err = db.Provider.UpdateUser(ctx, user)
	if err != nil {
		log.Debug("Failed to update user: ", err)
		return res, err
	}

	res = &model.Response{
		Message: `user access enabled successfully`,
	}

	go utils.RegisterEvent(ctx, constants.UserAccessEnabledWebhookEvent, "", user)

	return res, nil
}

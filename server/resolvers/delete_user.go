package resolvers

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// DeleteUserResolver is a resolver for delete user mutation
func DeleteUserResolver(ctx context.Context, params model.DeleteUserInput) (*model.Response, error) {
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
		"email": params.Email,
	})

	user, err := db.Provider.GetUserByEmail(params.Email)
	if err != nil {
		log.Debug("Failed to get user from DB: ", err)
		return res, err
	}

	go memorystore.Provider.DeleteAllUserSessions(user.ID)

	err = db.Provider.DeleteUser(user)
	if err != nil {
		log.Debug("Failed to delete user: ", err)
		return res, err
	}

	res = &model.Response{
		Message: `user deleted successfully`,
	}

	return res, nil
}

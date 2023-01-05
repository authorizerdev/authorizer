package resolvers

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// UserResolver is a resolver for user query
// This is admin only query
func UserResolver(ctx context.Context, params model.GetUserRequest) (*model.User, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin.")
		return nil, fmt.Errorf("unauthorized")
	}

	res, err := db.Provider.GetUserByID(ctx, params.ID)
	if err != nil {
		log.Debug("Failed to get users: ", err)
		return nil, err
	}

	return res.AsAPIUser(), nil
}

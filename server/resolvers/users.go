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

// UsersResolver is a resolver for users query
// This is admin only query
func UsersResolver(ctx context.Context, params *model.PaginatedInput) (*model.Users, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin.")
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(params)

	res, err := db.Provider.ListUsers(pagination)
	if err != nil {
		log.Debug("Failed to get users", err)
		return nil, err
	}

	return res, nil
}

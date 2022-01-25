package resolvers

import (
	"context"
	"fmt"

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
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(params)

	res, err := db.Provider.ListUsers(pagination)
	if err != nil {
		return nil, err
	}

	return res, nil
}

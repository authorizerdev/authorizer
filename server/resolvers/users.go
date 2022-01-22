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
func UsersResolver(ctx context.Context) ([]*model.User, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res []*model.User
	if err != nil {
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	users, err := db.Provider.ListUsers()
	if err != nil {
		return res, err
	}

	for i := 0; i < len(users); i++ {
		res = append(res, users[i].AsAPIUser())
	}

	return res, nil
}

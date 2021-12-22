package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

func Users(ctx context.Context) ([]*model.User, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res []*model.User
	if err != nil {
		return res, err
	}

	if !utils.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	users, err := db.Mgr.GetUsers()
	if err != nil {
		return res, err
	}

	for i := 0; i < len(users); i++ {
		res = append(res, utils.GetResUser(users[i]))
	}

	return res, nil
}

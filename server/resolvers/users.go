package resolvers

import (
	"context"
	"fmt"

	"github.com/yauthdev/yauth/server/db"
	"github.com/yauthdev/yauth/server/graph/model"
	"github.com/yauthdev/yauth/server/utils"
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

	for _, user := range users {
		res = append(res, &model.User{
			ID:              fmt.Sprintf("%d", user.ID),
			Email:           user.Email,
			SignupMethod:    user.SignupMethod,
			FirstName:       &user.FirstName,
			LastName:        &user.LastName,
			Password:        &user.Password,
			EmailVerifiedAt: &user.EmailVerifiedAt,
			CreatedAt:       &user.CreatedAt,
			UpdatedAt:       &user.UpdatedAt,
		})
	}

	return res, nil
}

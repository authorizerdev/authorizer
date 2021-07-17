package resolvers

import (
	"context"
	"fmt"

	"github.com/yauthdev/yauth/server/db"
	"github.com/yauthdev/yauth/server/graph/model"
)

func Users(ctx context.Context) ([]*model.User, error) {
	var res []*model.User
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
		})
	}

	return res, nil
}

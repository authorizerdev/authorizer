package resolvers

import (
	"context"
	"fmt"

	"github.com/yauthdev/yauth/server/db"
	"github.com/yauthdev/yauth/server/graph/model"
	"github.com/yauthdev/yauth/server/session"
	"github.com/yauthdev/yauth/server/utils"
)

func Profile(ctx context.Context) (*model.User, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.User
	if err != nil {
		return res, err
	}

	token, err := utils.GetAuthToken(gc)
	if err != nil {
		return res, err
	}

	claim, err := utils.VerifyAuthToken(token)
	if err != nil {
		return res, err
	}

	sessionToken := session.GetToken(claim.ID)

	if sessionToken == "" {
		return res, fmt.Errorf(`unauthorized`)
	}

	user, err := db.Mgr.GetUserByEmail(claim.Email)
	if err != nil {
		return res, err
	}

	userIdStr := fmt.Sprintf("%d", user.ID)

	res = &model.User{
		ID:              userIdStr,
		Email:           user.Email,
		Image:           &user.Image,
		FirstName:       &user.FirstName,
		LastName:        &user.LastName,
		SignupMethod:    user.SignupMethod,
		EmailVerifiedAt: &user.EmailVerifiedAt,
		CreatedAt:       &user.CreatedAt,
		UpdatedAt:       &user.UpdatedAt,
	}

	return res, nil
}

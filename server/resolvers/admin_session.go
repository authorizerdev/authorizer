package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

func AdminSession(ctx context.Context) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response

	if err != nil {
		return res, err
	}

	if !utils.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	hashedKey, err := utils.HashPassword(constants.EnvData.ADMIN_SECRET)
	if err != nil {
		return res, err
	}
	utils.SetAdminCookie(gc, hashedKey)

	res = &model.Response{
		Message: "admin logged in successfully",
	}
	return res, nil
}

package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

func AdminLoginResolver(ctx context.Context, params model.AdminLoginInput) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response

	if err != nil {
		return res, err
	}

	if params.AdminSecret != constants.EnvData.ADMIN_SECRET {
		return res, fmt.Errorf(`invalid admin secret`)
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

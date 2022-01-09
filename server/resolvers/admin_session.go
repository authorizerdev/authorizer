package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

func AdminSession(ctx context.Context) (*model.AdminLoginResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.AdminLoginResponse

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

	res = &model.AdminLoginResponse{
		AccessToken: hashedKey,
		Message:     "admin logged in successfully",
	}
	return res, nil
}
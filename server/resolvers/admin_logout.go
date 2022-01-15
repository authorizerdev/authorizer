package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

// AdminLogoutResolver is a resolver for admin logout mutation
func AdminLogoutResolver(ctx context.Context) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response

	if err != nil {
		return res, err
	}

	if !utils.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	utils.DeleteAdminCookie(gc)

	res = &model.Response{
		Message: "admin logged out successfully",
	}
	return res, nil
}

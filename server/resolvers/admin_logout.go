package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// AdminLogoutResolver is a resolver for admin logout mutation
func AdminLogoutResolver(ctx context.Context) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response

	if err != nil {
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	cookie.DeleteAdminCookie(gc)

	res = &model.Response{
		Message: "admin logged out successfully",
	}
	return res, nil
}

package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// AdminSessionResolver is a resolver for admin session query
func AdminSessionResolver(ctx context.Context) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response

	if err != nil {
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	hashedKey, err := utils.EncryptPassword(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
	if err != nil {
		return res, err
	}
	cookie.SetAdminCookie(gc, hashedKey)

	res = &model.Response{
		Message: "admin logged in successfully",
	}
	return res, nil
}

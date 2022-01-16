package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

// AdminLoginResolver is a resolver for admin login mutation
func AdminLoginResolver(ctx context.Context, params model.AdminLoginInput) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response

	if err != nil {
		return res, err
	}

	adminSecret := envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAdminSecret).(string)
	if params.AdminSecret != adminSecret {
		return res, fmt.Errorf(`invalid admin secret`)
	}

	hashedKey, err := utils.EncryptPassword(adminSecret)
	if err != nil {
		return res, err
	}
	utils.SetAdminCookie(gc, hashedKey)

	res = &model.Response{
		Message: "admin logged in successfully",
	}
	return res, nil
}

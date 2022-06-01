package resolvers

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/utils"
)

// AdminLoginResolver is a resolver for admin login mutation
func AdminLoginResolver(ctx context.Context, params model.AdminLoginInput) (*model.Response, error) {
	var res *model.Response

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
	if err != nil {
		log.Debug("Error getting admin secret: ", err)
		return res, err
	}
	if params.AdminSecret != adminSecret {
		log.Debug("Admin secret is not correct")
		return res, fmt.Errorf(`invalid admin secret`)
	}

	hashedKey, err := crypto.EncryptPassword(adminSecret)
	if err != nil {
		return res, err
	}
	cookie.SetAdminCookie(gc, hashedKey)

	res = &model.Response{
		Message: "admin logged in successfully",
	}
	return res, nil
}

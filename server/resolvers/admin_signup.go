package resolvers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

// AdminSignupResolver is a resolver for admin signup mutation
func AdminSignupResolver(ctx context.Context, params model.AdminSignupInput) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response

	if err != nil {
		return res, err
	}

	if strings.TrimSpace(params.AdminSecret) == "" {
		err = fmt.Errorf("please select secure admin secret")
		return res, err
	}

	if len(params.AdminSecret) < 6 {
		err = fmt.Errorf("admin secret must be at least 6 characters")
		return res, err
	}

	adminSecret := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)

	if adminSecret != "" {
		err = fmt.Errorf("admin sign up already completed")
		return res, err
	}

	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyAdminSecret, params.AdminSecret)
	// consvert EnvData to JSON
	var storeData envstore.Store

	jsonBytes, err := json.Marshal(envstore.EnvStoreObj.GetEnvStoreClone())
	if err != nil {
		return res, err
	}

	if err := json.Unmarshal(jsonBytes, &storeData); err != nil {
		return res, err
	}

	env, err := db.Provider.GetEnv()
	if err != nil {
		return res, err
	}

	envData, err := utils.EncryptEnvData(storeData)
	if err != nil {
		return res, err
	}

	env.EnvData = envData
	if _, err := db.Provider.UpdateEnv(env); err != nil {
		return res, err
	}

	hashedKey, err := utils.EncryptPassword(params.AdminSecret)
	if err != nil {
		return res, err
	}
	cookie.SetAdminCookie(gc, hashedKey)

	res = &model.Response{
		Message: "admin signed up successfully",
	}
	return res, nil
}

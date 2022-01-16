package resolvers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
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

	adminSecret := envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAdminSecret).(string)

	if adminSecret != "" {
		err = fmt.Errorf("admin sign up already completed")
		return res, err
	}

	envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.EnvKeyAdminSecret, params.AdminSecret)
	// consvert EnvData to JSON
	var jsonData map[string]interface{}

	jsonBytes, err := json.Marshal(envstore.EnvInMemoryStoreObj.GetEnvStoreClone())
	if err != nil {
		return res, err
	}

	if err := json.Unmarshal(jsonBytes, &jsonData); err != nil {
		return res, err
	}

	config, err := db.Mgr.GetConfig()
	if err != nil {
		return res, err
	}

	configData, err := utils.EncryptEnvData(jsonData)
	if err != nil {
		return res, err
	}

	config.Config = configData
	if _, err := db.Mgr.UpdateConfig(config); err != nil {
		return res, err
	}

	hashedKey, err := utils.EncryptPassword(params.AdminSecret)
	if err != nil {
		return res, err
	}
	utils.SetAdminCookie(gc, hashedKey)

	res = &model.Response{
		Message: "admin signed up successfully",
	}
	return res, nil
}

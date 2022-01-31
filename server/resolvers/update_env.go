package resolvers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// UpdateEnvResolver is a resolver for update config mutation
// This is admin only mutation
func UpdateEnvResolver(ctx context.Context, params model.UpdateEnvInput) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Response

	if err != nil {
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	var data map[string]interface{}
	byteData, err := json.Marshal(params)
	if err != nil {
		return res, fmt.Errorf("error marshalling params: %t", err)
	}

	err = json.Unmarshal(byteData, &data)
	if err != nil {
		return res, fmt.Errorf("error un-marshalling params: %t", err)
	}

	// in case of admin secret change update the cookie with new hash
	if params.AdminSecret != nil {
		if params.OldAdminSecret == nil {
			return res, errors.New("admin secret and old admin secret are required for secret change")
		}

		if *params.OldAdminSecret != envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret) {
			return res, errors.New("old admin secret is not correct")
		}

		if len(*params.AdminSecret) < 6 {
			err = fmt.Errorf("admin secret must be at least 6 characters")
			return res, err
		}

	}

	updatedData := envstore.EnvInMemoryStoreObj.GetEnvStoreClone()
	for key, value := range data {
		if value != nil {
			fieldType := reflect.TypeOf(value).String()

			if fieldType == "string" {
				updatedData.StringEnv[key] = value.(string)
			}

			if fieldType == "bool" {
				updatedData.BoolEnv[key] = value.(bool)
			}
			if fieldType == "[]interface {}" {
				stringArr := []string{}
				for _, v := range value.([]interface{}) {
					stringArr = append(stringArr, v.(string))
				}
				updatedData.SliceEnv[key] = stringArr
			}
		}
	}

	// handle derivative cases like disabling email verification & magic login
	// in case SMTP is off but env is set to true
	if updatedData.StringEnv[constants.EnvKeySmtpHost] == "" || updatedData.StringEnv[constants.EnvKeySmtpUsername] == "" || updatedData.StringEnv[constants.EnvKeySmtpPassword] == "" || updatedData.StringEnv[constants.EnvKeySenderEmail] == "" && updatedData.StringEnv[constants.EnvKeySmtpPort] == "" {
		if !updatedData.BoolEnv[constants.EnvKeyDisableEmailVerification] {
			updatedData.BoolEnv[constants.EnvKeyDisableEmailVerification] = true
		}

		if !updatedData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] {
			updatedData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = true
		}
	}

	// check the roles change
	if len(params.Roles) > 0 {
		if len(params.DefaultRoles) > 0 {
			// should be subset of roles
			for _, role := range params.DefaultRoles {
				if !utils.StringSliceContains(params.Roles, role) {
					return res, fmt.Errorf("default role %s is not in roles", role)
				}
			}
		}
	}

	if len(params.ProtectedRoles) > 0 {
		for _, role := range params.ProtectedRoles {
			if utils.StringSliceContains(params.Roles, role) || utils.StringSliceContains(params.DefaultRoles, role) {
				return res, fmt.Errorf("protected role %s found roles or default roles", role)
			}
		}
	}

	// Update local store
	envstore.EnvInMemoryStoreObj.UpdateEnvStore(updatedData)
	sessionstore.InitSession()
	oauth.InitOAuth()

	// Fetch the current db store and update it
	env, err := db.Provider.GetEnv()
	if err != nil {
		return res, err
	}

	if params.AdminSecret != nil {
		hashedKey, err := utils.EncryptPassword(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
		if err != nil {
			return res, err
		}
		cookie.SetAdminCookie(gc, hashedKey)
	}

	encryptedConfig, err := utils.EncryptEnvData(updatedData)
	if err != nil {
		return res, err
	}

	env.EnvData = encryptedConfig
	_, err = db.Provider.UpdateEnv(env)
	if err != nil {
		log.Println("error updating config:", err)
		return res, err
	}

	res = &model.Response{
		Message: "configurations updated successfully",
	}
	return res, nil
}

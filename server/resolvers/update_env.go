package resolvers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
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

	if !utils.IsSuperAdmin(gc) {
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

	updatedData := make(map[string]interface{})
	for key, value := range data {
		if value != nil {
			fieldType := reflect.TypeOf(value).String()

			if fieldType == "string" || fieldType == "bool" {
				updatedData[key] = value
			}

			if fieldType == "[]interface {}" {
				stringArr := []string{}
				for _, v := range value.([]interface{}) {
					stringArr = append(stringArr, v.(string))
				}
				updatedData[key] = stringArr
			}
		}
	}

	// handle derivative cases like disabling email verification & magic login
	// in case SMTP is off but env is set to true
	if updatedData[constants.EnvKeySmtpHost] == "" || updatedData[constants.EnvKeySenderEmail] == "" || updatedData[constants.EnvKeySmtpPort] == "" || updatedData[constants.EnvKeySmtpUsername] == "" || updatedData[constants.EnvKeySmtpPassword] == "" {
		if !updatedData[constants.EnvKeyDisableEmailVerification].(bool) {
			updatedData[constants.EnvKeyDisableEmailVerification] = true
		}

		if !updatedData[constants.EnvKeyDisableMagicLinkLogin].(bool) {
			updatedData[constants.EnvKeyDisableMagicLinkLogin] = true
		}
	}

	config, err := db.Mgr.GetConfig()
	if err != nil {
		return res, err
	}

	envstore.EnvInMemoryStoreObj.UpdateEnvStore(updatedData)

	encryptedConfig, err := utils.EncryptEnvData(updatedData)
	if err != nil {
		return res, err
	}

	// in case of db change re-initialize db
	if params.DatabaseType != nil || params.DatabaseURL != nil || params.DatabaseName != nil {
		db.InitDB()
	}

	// in case of admin secret change update the cookie with new hash
	if params.AdminSecret != nil {
		hashedKey, err := utils.EncryptPassword(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAdminSecret).(string))
		if err != nil {
			return res, err
		}
		utils.SetAdminCookie(gc, hashedKey)
	}

	config.Config = encryptedConfig
	_, err = db.Mgr.UpdateConfig(config)
	if err != nil {
		log.Println("error updating config:", err)
		return res, err
	}

	res = &model.Response{
		Message: "configurations updated successfully",
	}
	return res, nil
}

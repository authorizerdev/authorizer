package resolvers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

func UpdateConfigResolver(ctx context.Context, params model.UpdateConfigInput) (*model.Response, error) {
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
	if updatedData["SMTP_HOST"] == "" || updatedData["SENDER_EMAIL"] == "" || updatedData["SENDER_PASSWORD"] == "" {
		if !updatedData["DISABLE_EMAIL_VERIFICATION"].(bool) {
			updatedData["DISABLE_EMAIL_VERIFICATION"] = true
		}

		if !updatedData["DISABLE_MAGIC_LINK_LOGIN"].(bool) {
			updatedData["DISABLE_MAGIC_LINK_LOGIN"] = true
		}
	}

	config, err := db.Mgr.GetConfig()
	if err != nil {
		return res, err
	}

	encryptedConfig, err := utils.EncryptConfig(updatedData)
	if err != nil {
		return res, err
	}

	// in case of db change re-initialize db
	if params.DatabaseType != nil || params.DatabaseURL != nil || params.DatabaseName != nil {
		db.InitDB()
	}

	// in case of admin secret change update the cookie with new hash
	if params.AdminSecret != nil {
		hashedKey, err := utils.HashPassword(constants.EnvData.ADMIN_SECRET)
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

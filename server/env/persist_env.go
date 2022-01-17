package env

import (
	"encoding/json"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/google/uuid"
)

func PersistEnv() error {
	config, err := db.Mgr.GetConfig()
	// config not found in db
	if err != nil {
		// AES encryption needs 32 bit key only, so we chop off last 4 characters from 36 bit uuid
		hash := uuid.New().String()[:36-4]
		constants.EnvData.ENCRYPTION_KEY = hash
		encodedHash := utils.EncryptB64(hash)

		configData, err := json.Marshal(constants.EnvData)
		if err != nil {
			return err
		}
		encryptedConfig, err := utils.EncryptAES(configData)
		if err != nil {
			return err
		}

		config = db.Config{
			Hash:   encodedHash,
			Config: encryptedConfig,
		}

		db.Mgr.AddConfig(config)
	} else {
		// decrypt the config data from db
		// decryption can be done using the hash stored in db
		encryptionKey := config.Hash
		decryptedEncryptionKey, err := utils.DecryptB64(encryptionKey)
		if err != nil {
			return err
		}
		constants.EnvData.ENCRYPTION_KEY = decryptedEncryptionKey
		decryptedConfigs, err := utils.DecryptAES(config.Config)
		if err != nil {
			return err
		}

		// temp json to validate with env
		var jsonData map[string]interface{}

		err = json.Unmarshal(decryptedConfigs, &jsonData)
		if err != nil {
			return err
		}

		// if env is changed via env file or OS env
		// give that higher preference and update db, but we don't recommend it

		hasChanged := false

		for key, value := range jsonData {
			fieldType := reflect.TypeOf(value).String()

			// check only for derivative keys
			// No need to check for ENCRYPTION_KEY which special key we use for encrypting config data
			// as we have removed it from json
			envValue := strings.TrimSpace(os.Getenv(key))

			// env is not empty
			if envValue != "" {
				// check the type
				// currently we have 3 types of env vars: string, bool, []string{}
				if fieldType == "string" {
					if value != envValue {
						jsonData[key] = envValue
						hasChanged = true
					}
				}

				if fieldType == "bool" {
					newValue := envValue == "true"
					if value != newValue {
						jsonData[key] = newValue
						hasChanged = true
					}
				}

				if fieldType == "[]interface {}" {
					stringArr := []string{}
					envStringArr := strings.Split(envValue, ",")
					for _, v := range value.([]interface{}) {
						stringArr = append(stringArr, v.(string))
					}
					if !utils.IsStringArrayEqual(stringArr, envStringArr) {
						jsonData[key] = envStringArr
					}
				}
			}
		}

		// handle derivative cases like disabling email verification & magic login
		// in case SMTP is off but env is set to true
		if jsonData["SMTP_HOST"] == "" || jsonData["SENDER_EMAIL"] == "" || jsonData["SENDER_PASSWORD"] == "" {
			if !jsonData["DISABLE_EMAIL_VERIFICATION"].(bool) {
				jsonData["DISABLE_EMAIL_VERIFICATION"] = true
				hasChanged = true
			}

			if !jsonData["DISABLE_MAGIC_LINK_LOGIN"].(bool) {
				jsonData["DISABLE_MAGIC_LINK_LOGIN"] = true
				hasChanged = true
			}
		}

		if hasChanged {
			encryptedConfig, err := utils.EncryptConfig(jsonData)
			if err != nil {
				return err
			}

			config.Config = encryptedConfig
			_, err = db.Mgr.UpdateConfig(config)
			if err != nil {
				log.Println("error updating config:", err)
				return err
			}
		}

	}

	return nil
}

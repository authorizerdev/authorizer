package env

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/utils"
)

func fixBackwardCompatibility(data map[string]interface{}) (bool, map[string]interface{}) {
	result := data
	// check if env data is stored in older format
	hasOlderFormat := false
	if _, ok := result["bool_env"]; ok {
		for key, value := range result["bool_env"].(map[string]interface{}) {
			result[key] = value
		}
		hasOlderFormat = true
		delete(result, "bool_env")
	}

	if _, ok := result["string_env"]; ok {
		for key, value := range result["string_env"].(map[string]interface{}) {
			result[key] = value
		}
		hasOlderFormat = true
		delete(result, "string_env")
	}

	if _, ok := result["slice_env"]; ok {
		for key, value := range result["slice_env"].(map[string]interface{}) {
			typeOfValue := reflect.TypeOf(value)
			if strings.Contains(typeOfValue.String(), "[]string") {
				result[key] = strings.Join(value.([]string), ",")
			}
			if strings.Contains(typeOfValue.String(), "[]interface") {
				result[key] = strings.Join(utils.ConvertInterfaceToStringSlice(value), ",")
			}
		}
		hasOlderFormat = true
		delete(result, "slice_env")
	}

	return hasOlderFormat, result
}

// GetEnvData returns the env data from database
func GetEnvData() (map[string]interface{}, error) {
	var result map[string]interface{}
	env, err := db.Provider.GetEnv()
	// config not found in db
	if err != nil {
		log.Debug("Error while getting env data from db: ", err)
		return result, err
	}

	encryptionKey := env.Hash
	decryptedEncryptionKey, err := crypto.DecryptB64(encryptionKey)
	if err != nil {
		log.Debug("Error while decrypting encryption key: ", err)
		return result, err
	}

	memorystore.Provider.UpdateEnvVariable(constants.EnvKeyEncryptionKey, decryptedEncryptionKey)

	b64DecryptedConfig, err := crypto.DecryptB64(env.EnvData)
	if err != nil {
		log.Debug("Error while decrypting env data from B64: ", err)
		return result, err
	}

	decryptedConfigs, err := crypto.DecryptAESEnv([]byte(b64DecryptedConfig))
	if err != nil {
		log.Debug("Error while decrypting env data from AES: ", err)
		return result, err
	}

	err = json.Unmarshal(decryptedConfigs, &result)
	if err != nil {
		log.Debug("Error while unmarshalling env data: ", err)
		return result, err
	}

	hasOlderFormat, result := fixBackwardCompatibility(result)

	if hasOlderFormat {
		err = memorystore.Provider.UpdateEnvStore(result)
		if err != nil {
			log.Debug("Error while updating env store: ", err)
			return result, err
		}

	}

	return result, err
}

// PersistEnv persists the environment variables to the database
func PersistEnv() error {
	env, err := db.Provider.GetEnv()
	// config not found in db
	if err != nil {
		// AES encryption needs 32 bit key only, so we chop off last 4 characters from 36 bit uuid
		hash := uuid.New().String()[:36-4]
		err := memorystore.Provider.UpdateEnvVariable(constants.EnvKeyEncryptionKey, hash)
		if err != nil {
			log.Debug("Error while updating encryption env variable: ", err)
			return err
		}
		encodedHash := crypto.EncryptB64(hash)

		res, err := memorystore.Provider.GetEnvStore()
		if err != nil {
			log.Debug("Error while getting env store: ", err)
			return err
		}

		encryptedConfig, err := crypto.EncryptEnvData(res)
		if err != nil {
			log.Debug("Error while encrypting env data: ", err)
			return err
		}

		env = models.Env{
			Hash:    encodedHash,
			EnvData: encryptedConfig,
		}

		env, err = db.Provider.AddEnv(env)
		if err != nil {
			log.Debug("Error while persisting env data to db: ", err)
			return err
		}
	} else {
		// decrypt the config data from db
		// decryption can be done using the hash stored in db
		encryptionKey := env.Hash
		decryptedEncryptionKey, err := crypto.DecryptB64(encryptionKey)
		if err != nil {
			log.Debug("Error while decrypting encryption key: ", err)
			return err
		}

		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyEncryptionKey, decryptedEncryptionKey)

		b64DecryptedConfig, err := crypto.DecryptB64(env.EnvData)
		if err != nil {
			log.Debug("Error while decrypting env data from B64: ", err)
			return err
		}

		decryptedConfigs, err := crypto.DecryptAESEnv([]byte(b64DecryptedConfig))
		if err != nil {
			log.Debug("Error while decrypting env data from AES: ", err)
			return err
		}

		// temp store variable
		storeData := map[string]interface{}{}

		err = json.Unmarshal(decryptedConfigs, &storeData)
		if err != nil {
			log.Debug("Error while unmarshalling env data: ", err)
			return err
		}

		hasOlderFormat, result := fixBackwardCompatibility(storeData)
		if hasOlderFormat {
			err = memorystore.Provider.UpdateEnvStore(result)
			if err != nil {
				log.Debug("Error while updating env store: ", err)
				return err
			}

		}

		// if env is changed via env file or OS env
		// give that higher preference and update db, but we don't recommend it

		hasChanged := false
		for key, value := range storeData {
			// don't override unexposed envs
			// check only for derivative keys
			// No need to check for ENCRYPTION_KEY which special key we use for encrypting config data
			// as we have removed it from json
			if key != constants.EnvKeyEncryptionKey {
				envValue := strings.TrimSpace(os.Getenv(key))
				if envValue != "" {
					switch key {
					case constants.EnvKeyIsProd, constants.EnvKeyDisableBasicAuthentication, constants.EnvKeyDisableEmailVerification, constants.EnvKeyDisableLoginPage, constants.EnvKeyDisableMagicLinkLogin, constants.EnvKeyDisableSignUp:
						if envValueBool, err := strconv.ParseBool(envValue); err == nil {
							if value.(bool) != envValueBool {
								storeData[key] = envValueBool
								hasChanged = true
							}
						}
					default:
						if value != nil && value.(string) != envValue {
							storeData[key] = envValue
							hasChanged = true
						}
					}
				}
			}
		}

		// handle derivative cases like disabling email verification & magic login
		// in case SMTP is off but env is set to true
		if storeData[constants.EnvKeySmtpHost] == "" || storeData[constants.EnvKeySmtpUsername] == "" || storeData[constants.EnvKeySmtpPassword] == "" || storeData[constants.EnvKeySenderEmail] == "" && storeData[constants.EnvKeySmtpPort] == "" {
			if !storeData[constants.EnvKeyDisableEmailVerification].(bool) {
				storeData[constants.EnvKeyDisableEmailVerification] = true
				hasChanged = true
			}

			if !storeData[constants.EnvKeyDisableMagicLinkLogin].(bool) {
				storeData[constants.EnvKeyDisableMagicLinkLogin] = true
				hasChanged = true
			}
		}

		err = memorystore.Provider.UpdateEnvStore(storeData)
		if err != nil {
			log.Debug("Error while updating env store: ", err)
			return err
		}

		jwk, err := crypto.GenerateJWKBasedOnEnv()
		if err != nil {
			log.Debug("Error while generating JWK: ", err)
			return err
		}
		// updating jwk
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyJWK, jwk)

		if hasChanged {
			encryptedConfig, err := crypto.EncryptEnvData(storeData)
			if err != nil {
				log.Debug("Error while encrypting env data: ", err)
				return err
			}

			env.EnvData = encryptedConfig
			_, err = db.Provider.UpdateEnv(env)
			if err != nil {
				log.Debug("Failed to Update Config: ", err)
				return err
			}
		}
	}

	return nil
}

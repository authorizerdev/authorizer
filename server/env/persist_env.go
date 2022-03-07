package env

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/utils"
)

// GetEnvData returns the env data from database
func GetEnvData() (envstore.Store, error) {
	var result envstore.Store
	env, err := db.Provider.GetEnv()
	// config not found in db
	if err != nil {
		return result, err
	}

	encryptionKey := env.Hash
	decryptedEncryptionKey, err := crypto.DecryptB64(encryptionKey)
	if err != nil {
		return result, err
	}

	envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyEncryptionKey, decryptedEncryptionKey)

	b64DecryptedConfig, err := crypto.DecryptB64(env.EnvData)
	if err != nil {
		return result, err
	}

	decryptedConfigs, err := crypto.DecryptAESEnv([]byte(b64DecryptedConfig))
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(decryptedConfigs, &result)
	if err != nil {
		return result, err
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
		envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyEncryptionKey, hash)
		encodedHash := crypto.EncryptB64(hash)

		encryptedConfig, err := crypto.EncryptEnvData(envstore.EnvStoreObj.GetEnvStoreClone())
		if err != nil {
			return err
		}

		env = models.Env{
			Hash:    encodedHash,
			EnvData: encryptedConfig,
		}

		env, err = db.Provider.AddEnv(env)
		if err != nil {
			return err
		}
	} else {
		// decrypt the config data from db
		// decryption can be done using the hash stored in db
		encryptionKey := env.Hash
		decryptedEncryptionKey, err := crypto.DecryptB64(encryptionKey)
		if err != nil {
			return err
		}

		envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyEncryptionKey, decryptedEncryptionKey)

		b64DecryptedConfig, err := crypto.DecryptB64(env.EnvData)
		if err != nil {
			return err
		}

		decryptedConfigs, err := crypto.DecryptAESEnv([]byte(b64DecryptedConfig))
		if err != nil {
			return err
		}

		// temp store variable
		var storeData envstore.Store

		err = json.Unmarshal(decryptedConfigs, &storeData)
		if err != nil {
			return err
		}

		// if env is changed via env file or OS env
		// give that higher preference and update db, but we don't recommend it

		hasChanged := false

		for key, value := range storeData.StringEnv {
			// don't override unexposed envs
			if key != constants.EnvKeyEncryptionKey && key != constants.EnvKeyClientID && key != constants.EnvKeyClientSecret && key != constants.EnvKeyJWK {
				// check only for derivative keys
				// No need to check for ENCRYPTION_KEY which special key we use for encrypting config data
				// as we have removed it from json
				envValue := strings.TrimSpace(os.Getenv(key))

				// env is not empty
				if envValue != "" {
					if value != envValue {
						storeData.StringEnv[key] = envValue
						hasChanged = true
					}
				}
			}
		}

		for key, value := range storeData.BoolEnv {
			envValue := strings.TrimSpace(os.Getenv(key))
			// env is not empty
			if envValue != "" {
				envValueBool, _ := strconv.ParseBool(envValue)
				if value != envValueBool {
					storeData.BoolEnv[key] = envValueBool
					hasChanged = true
				}
			}
		}

		for key, value := range storeData.SliceEnv {
			envValue := strings.TrimSpace(os.Getenv(key))
			// env is not empty
			if envValue != "" {
				envStringArr := strings.Split(envValue, ",")
				if !utils.IsStringArrayEqual(value, envStringArr) {
					storeData.SliceEnv[key] = envStringArr
					hasChanged = true
				}
			}
		}

		// handle derivative cases like disabling email verification & magic login
		// in case SMTP is off but env is set to true
		if storeData.StringEnv[constants.EnvKeySmtpHost] == "" || storeData.StringEnv[constants.EnvKeySmtpUsername] == "" || storeData.StringEnv[constants.EnvKeySmtpPassword] == "" || storeData.StringEnv[constants.EnvKeySenderEmail] == "" && storeData.StringEnv[constants.EnvKeySmtpPort] == "" {
			if !storeData.BoolEnv[constants.EnvKeyDisableEmailVerification] {
				storeData.BoolEnv[constants.EnvKeyDisableEmailVerification] = true
				hasChanged = true
			}

			if !storeData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] {
				storeData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = true
				hasChanged = true
			}
		}
		envstore.EnvStoreObj.UpdateEnvStore(storeData)
		jwk, err := crypto.GenerateJWKBasedOnEnv()
		if err != nil {
			return err
		}
		// updating jwk
		envstore.EnvStoreObj.UpdateEnvVariable(constants.StringStoreIdentifier, constants.EnvKeyJWK, jwk)

		if hasChanged {
			encryptedConfig, err := crypto.EncryptEnvData(storeData)
			if err != nil {
				return err
			}

			env.EnvData = encryptedConfig
			_, err = db.Provider.UpdateEnv(env)
			if err != nil {
				log.Println("error updating config:", err)
				return err
			}
		}
	}

	return nil
}

package utils

import (
	"encoding/json"

	"github.com/authorizerdev/authorizer/server/envstore"
)

func EncryptConfig(data map[string]interface{}) ([]byte, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return []byte{}, err
	}

	envData := envstore.EnvInMemoryStoreObj.GetEnvStoreClone()

	err = json.Unmarshal(jsonBytes, &envData)
	if err != nil {
		return []byte{}, err
	}

	configData, err := json.Marshal(envData)
	if err != nil {
		return []byte{}, err
	}
	encryptedConfig, err := EncryptAES(configData)
	if err != nil {
		return []byte{}, err
	}

	return encryptedConfig, nil
}

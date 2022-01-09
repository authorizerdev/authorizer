package utils

import (
	"encoding/json"

	"github.com/authorizerdev/authorizer/server/constants"
)

func EncryptConfig(data map[string]interface{}) ([]byte, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return []byte{}, err
	}

	err = json.Unmarshal(jsonBytes, &constants.EnvData)
	if err != nil {
		return []byte{}, err
	}

	configData, err := json.Marshal(constants.EnvData)
	if err != nil {
		return []byte{}, err
	}
	encryptedConfig, err := EncryptAES(configData)
	if err != nil {
		return []byte{}, err
	}

	return encryptedConfig, nil
}

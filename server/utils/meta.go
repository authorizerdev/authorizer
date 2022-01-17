package utils

import (
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
)

// GetMeta helps in getting the meta data about the deployment from EnvData
func GetMetaInfo() model.Meta {
	return model.Meta{
		Version:                      envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyVersion).(string),
		IsGoogleLoginEnabled:         envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGoogleClientID).(string) != "" && envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGoogleClientSecret).(string) != "",
		IsGithubLoginEnabled:         envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGithubClientID).(string) != "" && envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyGithubClientSecret).(string) != "",
		IsFacebookLoginEnabled:       envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyFacebookClientID).(string) != "" && envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyFacebookClientSecret).(string) != "",
		IsBasicAuthenticationEnabled: !envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDisableBasicAuthentication).(bool),
		IsEmailVerificationEnabled:   !envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDisableEmailVerification).(bool),
		IsMagicLinkLoginEnabled:      !envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyDisableMagicLinkLogin).(bool),
	}
}

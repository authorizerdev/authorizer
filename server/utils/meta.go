package utils

import (
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
)

// GetMeta helps in getting the meta data about the deployment from EnvData
func GetMetaInfo() model.Meta {
	return model.Meta{
		Version:                      envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyVersion),
		ClientID:                     envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID),
		IsGoogleLoginEnabled:         envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientID) != "" && envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientSecret) != "",
		IsGithubLoginEnabled:         envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientID) != "" && envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientSecret) != "",
		IsFacebookLoginEnabled:       envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientID) != "" && envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientSecret) != "",
		IsBasicAuthenticationEnabled: !envstore.EnvInMemoryStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication),
		IsEmailVerificationEnabled:   !envstore.EnvInMemoryStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification),
		IsMagicLinkLoginEnabled:      !envstore.EnvInMemoryStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableMagicLinkLogin),
	}
}

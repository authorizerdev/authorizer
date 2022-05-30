package utils

import (
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
)

// GetMeta helps in getting the meta data about the deployment from EnvData
func GetMetaInfo() model.Meta {
	return model.Meta{
		Version:                      constants.VERSION,
		ClientID:                     memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyClientID),
		IsGoogleLoginEnabled:         memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientID) != "" && memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientSecret) != "",
		IsGithubLoginEnabled:         memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGithubClientID) != "" && memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyGithubClientSecret) != "",
		IsFacebookLoginEnabled:       memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientID) != "" && memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientSecret) != "",
		IsBasicAuthenticationEnabled: !memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication),
		IsEmailVerificationEnabled:   !memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification),
		IsMagicLinkLoginEnabled:      !memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableMagicLinkLogin),
		IsSignUpEnabled:              !memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableSignUp),
	}
}

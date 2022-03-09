package utils

import (
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
)

// GetMeta helps in getting the meta data about the deployment from EnvData
func GetMetaInfo() model.Meta {
	return model.Meta{
		Version:                      envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyVersion),
		ClientID:                     envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyClientID),
		IsGoogleLoginEnabled:         envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientID) != "" && envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGoogleClientSecret) != "",
		IsGithubLoginEnabled:         envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientID) != "" && envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyGithubClientSecret) != "",
		IsFacebookLoginEnabled:       envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientID) != "" && envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyFacebookClientSecret) != "",
		IsBasicAuthenticationEnabled: !envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableBasicAuthentication),
		IsEmailVerificationEnabled:   !envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableEmailVerification),
		IsMagicLinkLoginEnabled:      !envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableMagicLinkLogin),
	}
}

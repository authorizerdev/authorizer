package utils

import (
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
)

// GetMeta helps in getting the meta data about the deployment
// version,
func GetMetaInfo() model.Meta {
	return model.Meta{
		Version:                      constants.EnvData.VERSION,
		IsGoogleLoginEnabled:         constants.EnvData.GOOGLE_CLIENT_ID != "" && constants.EnvData.GOOGLE_CLIENT_SECRET != "",
		IsGithubLoginEnabled:         constants.EnvData.GITHUB_CLIENT_ID != "" && constants.EnvData.GOOGLE_CLIENT_SECRET != "",
		IsFacebookLoginEnabled:       constants.EnvData.FACEBOOK_CLIENT_ID != "" && constants.EnvData.FACEBOOK_CLIENT_SECRET != "",
		IsBasicAuthenticationEnabled: !constants.EnvData.DISABLE_BASIC_AUTHENTICATION,
		IsEmailVerificationEnabled:   !constants.EnvData.DISABLE_EMAIL_VERIFICATION,
		IsMagicLinkLoginEnabled:      !constants.EnvData.DISABLE_MAGIC_LINK_LOGIN,
	}
}

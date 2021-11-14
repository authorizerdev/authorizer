package utils

import (
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
)

// GetMeta helps in getting the meta data about the deployment
// version,
func GetMetaInfo() model.Meta {
	return model.Meta{
		Version:                      constants.VERSION,
		IsGoogleLoginEnabled:         constants.GOOGLE_CLIENT_ID != "" && constants.GOOGLE_CLIENT_SECRET != "",
		IsGithubLoginEnabled:         constants.GITHUB_CLIENT_ID != "" && constants.GOOGLE_CLIENT_SECRET != "",
		IsFacebookLoginEnabled:       constants.FACEBOOK_CLIENT_ID != "" && constants.FACEBOOK_CLIENT_SECRET != "",
		IsTwitterLoginEnabled:        constants.TWITTER_CLIENT_ID != "" && constants.TWITTER_CLIENT_SECRET != "",
		IsBasicAuthenticationEnabled: constants.DISABLE_BASIC_AUTHENTICATION != "true",
		IsEmailVerificationEnabled:   constants.DISABLE_EMAIL_VERIFICATION != "true",
		IsMagicLoginEnabled:          constants.DISABLE_MAGIC_LOGIN != "true" && constants.DISABLE_EMAIL_VERIFICATION != "true",
	}
}

package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

// TODO rename to env_data

// ConfigResolver is a resolver for config query
// This is admin only query
func ConfigResolver(ctx context.Context) (*model.Config, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Config

	if err != nil {
		return res, err
	}

	if !utils.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	res = &model.Config{
		AdminSecret:                &constants.EnvData.ADMIN_SECRET,
		DatabaseType:               &constants.EnvData.DATABASE_TYPE,
		DatabaseURL:                &constants.EnvData.DATABASE_URL,
		DatabaseName:               &constants.EnvData.DATABASE_NAME,
		SMTPHost:                   &constants.EnvData.SMTP_HOST,
		SMTPPort:                   &constants.EnvData.SMTP_PORT,
		SMTPPassword:               &constants.EnvData.SMTP_PASSWORD,
		SMTPUsername:               &constants.EnvData.SMTP_USERNAME,
		SenderEmail:                &constants.EnvData.SENDER_EMAIL,
		JwtType:                    &constants.EnvData.JWT_TYPE,
		JwtSecret:                  &constants.EnvData.JWT_SECRET,
		AllowedOrigins:             constants.EnvData.ALLOWED_ORIGINS,
		AuthorizerURL:              &constants.EnvData.AUTHORIZER_URL,
		AppURL:                     &constants.EnvData.APP_URL,
		RedisURL:                   &constants.EnvData.REDIS_URL,
		CookieName:                 &constants.EnvData.COOKIE_NAME,
		ResetPasswordURL:           &constants.EnvData.RESET_PASSWORD_URL,
		DisableEmailVerification:   &constants.EnvData.DISABLE_EMAIL_VERIFICATION,
		DisableBasicAuthentication: &constants.EnvData.DISABLE_BASIC_AUTHENTICATION,
		DisableMagicLinkLogin:      &constants.EnvData.DISABLE_MAGIC_LINK_LOGIN,
		DisableLoginPage:           &constants.EnvData.DISABLE_LOGIN_PAGE,
		Roles:                      constants.EnvData.ROLES,
		ProtectedRoles:             constants.EnvData.PROTECTED_ROLES,
		DefaultRoles:               constants.EnvData.DEFAULT_ROLES,
		JwtRoleClaim:               &constants.EnvData.JWT_ROLE_CLAIM,
		GoogleClientID:             &constants.EnvData.GOOGLE_CLIENT_ID,
		GoogleClientSecret:         &constants.EnvData.GOOGLE_CLIENT_SECRET,
		GithubClientID:             &constants.EnvData.GITHUB_CLIENT_ID,
		GithubClientSecret:         &constants.EnvData.GITHUB_CLIENT_SECRET,
		FacebookClientID:           &constants.EnvData.FACEBOOK_CLIENT_ID,
		FacebookClientSecret:       &constants.EnvData.FACEBOOK_CLIENT_SECRET,
		OrganizationName:           &constants.EnvData.ORGANIZATION_NAME,
		OrganizationLogo:           &constants.EnvData.ORGANIZATION_LOGO,
	}
	return res, nil
}

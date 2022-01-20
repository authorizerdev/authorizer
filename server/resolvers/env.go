package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

// EnvResolver is a resolver for config query
// This is admin only query
func EnvResolver(ctx context.Context) (*model.Env, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.Env

	if err != nil {
		return res, err
	}

	if !utils.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	// get clone of store
	store := envstore.EnvInMemoryStoreObj.GetEnvStoreClone()
	adminSecret := store.StringEnv[constants.EnvKeyAdminSecret]
	databaseType := store.StringEnv[constants.EnvKeyDatabaseType]
	databaseURL := store.StringEnv[constants.EnvKeyDatabaseURL]
	databaseName := store.StringEnv[constants.EnvKeyDatabaseName]
	smtpHost := store.StringEnv[constants.EnvKeySmtpHost]
	smtpPort := store.StringEnv[constants.EnvKeySmtpPort]
	smtpUsername := store.StringEnv[constants.EnvKeySmtpUsername]
	smtpPassword := store.StringEnv[constants.EnvKeySmtpPassword]
	senderEmail := store.StringEnv[constants.EnvKeySenderEmail]
	jwtType := store.StringEnv[constants.EnvKeyJwtType]
	jwtSecret := store.StringEnv[constants.EnvKeyJwtSecret]
	jwtRoleClaim := store.StringEnv[constants.EnvKeyJwtRoleClaim]
	allowedOrigins := store.SliceEnv[constants.EnvKeyAllowedOrigins]
	authorizerURL := store.StringEnv[constants.EnvKeyAuthorizerURL]
	appURL := store.StringEnv[constants.EnvKeyAppURL]
	redisURL := store.StringEnv[constants.EnvKeyRedisURL]
	cookieName := store.StringEnv[constants.EnvKeyCookieName]
	resetPasswordURL := store.StringEnv[constants.EnvKeyResetPasswordURL]
	disableEmailVerification := store.BoolEnv[constants.EnvKeyDisableEmailVerification]
	disableBasicAuthentication := store.BoolEnv[constants.EnvKeyDisableBasicAuthentication]
	disableMagicLinkLogin := store.BoolEnv[constants.EnvKeyDisableMagicLinkLogin]
	disableLoginPage := store.BoolEnv[constants.EnvKeyDisableLoginPage]
	roles := store.SliceEnv[constants.EnvKeyRoles]
	defaultRoles := store.SliceEnv[constants.EnvKeyDefaultRoles]
	protectedRoles := store.SliceEnv[constants.EnvKeyProtectedRoles]
	googleClientID := store.StringEnv[constants.EnvKeyGoogleClientID]
	googleClientSecret := store.StringEnv[constants.EnvKeyGoogleClientSecret]
	facebookClientID := store.StringEnv[constants.EnvKeyFacebookClientID]
	facebookClientSecret := store.StringEnv[constants.EnvKeyFacebookClientSecret]
	githubClientID := store.StringEnv[constants.EnvKeyGithubClientID]
	githubClientSecret := store.StringEnv[constants.EnvKeyGithubClientSecret]
	organizationName := store.StringEnv[constants.EnvKeyOrganizationName]
	organizationLogo := store.StringEnv[constants.EnvKeyOrganizationLogo]

	res = &model.Env{
		AdminSecret:                &adminSecret,
		DatabaseType:               &databaseType,
		DatabaseURL:                &databaseURL,
		DatabaseName:               &databaseName,
		SMTPHost:                   &smtpHost,
		SMTPPort:                   &smtpPort,
		SMTPPassword:               &smtpPassword,
		SMTPUsername:               &smtpUsername,
		SenderEmail:                &senderEmail,
		JwtType:                    &jwtType,
		JwtSecret:                  &jwtSecret,
		JwtRoleClaim:               &jwtRoleClaim,
		AllowedOrigins:             allowedOrigins,
		AuthorizerURL:              &authorizerURL,
		AppURL:                     &appURL,
		RedisURL:                   &redisURL,
		CookieName:                 &cookieName,
		ResetPasswordURL:           &resetPasswordURL,
		DisableEmailVerification:   &disableEmailVerification,
		DisableBasicAuthentication: &disableBasicAuthentication,
		DisableMagicLinkLogin:      &disableMagicLinkLogin,
		DisableLoginPage:           &disableLoginPage,
		Roles:                      roles,
		ProtectedRoles:             protectedRoles,
		DefaultRoles:               defaultRoles,
		GoogleClientID:             &googleClientID,
		GoogleClientSecret:         &googleClientSecret,
		GithubClientID:             &githubClientID,
		GithubClientSecret:         &githubClientSecret,
		FacebookClientID:           &facebookClientID,
		FacebookClientSecret:       &facebookClientSecret,
		OrganizationName:           &organizationName,
		OrganizationLogo:           &organizationLogo,
	}
	return res, nil
}

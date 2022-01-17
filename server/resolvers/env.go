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
	adminSecret := store[constants.EnvKeyAdminSecret].(string)
	databaseType := store[constants.EnvKeyDatabaseType].(string)
	databaseURL := store[constants.EnvKeyDatabaseURL].(string)
	databaseName := store[constants.EnvKeyDatabaseName].(string)
	smtpHost := store[constants.EnvKeySmtpHost].(string)
	smtpPort := store[constants.EnvKeySmtpPort].(string)
	smtpUsername := store[constants.EnvKeySmtpUsername].(string)
	smtpPassword := store[constants.EnvKeySmtpPassword].(string)
	senderEmail := store[constants.EnvKeySenderEmail].(string)
	jwtType := store[constants.EnvKeyJwtType].(string)
	jwtSecret := store[constants.EnvKeyJwtSecret].(string)
	jwtRoleClaim := store[constants.EnvKeyJwtRoleClaim].(string)
	allowedOrigins := store[constants.EnvKeyAllowedOrigins].([]string)
	authorizerURL := store[constants.EnvKeyAuthorizerURL].(string)
	appURL := store[constants.EnvKeyAppURL].(string)
	redisURL := store[constants.EnvKeyRedisURL].(string)
	cookieName := store[constants.EnvKeyCookieName].(string)
	resetPasswordURL := store[constants.EnvKeyResetPasswordURL].(string)
	disableEmailVerification := store[constants.EnvKeyDisableEmailVerification].(bool)
	disableBasicAuthentication := store[constants.EnvKeyDisableBasicAuthentication].(bool)
	disableMagicLinkLogin := store[constants.EnvKeyDisableMagicLinkLogin].(bool)
	disableLoginPage := store[constants.EnvKeyDisableLoginPage].(bool)
	roles := store[constants.EnvKeyRoles].([]string)
	defaultRoles := store[constants.EnvKeyDefaultRoles].([]string)
	protectedRoles := store[constants.EnvKeyProtectedRoles].([]string)
	googleClientID := store[constants.EnvKeyGoogleClientID].(string)
	googleClientSecret := store[constants.EnvKeyGoogleClientSecret].(string)
	facebookClientID := store[constants.EnvKeyFacebookClientID].(string)
	facebookClientSecret := store[constants.EnvKeyFacebookClientSecret].(string)
	githubClientID := store[constants.EnvKeyGithubClientID].(string)
	githubClientSecret := store[constants.EnvKeyGithubClientSecret].(string)
	organizationName := store[constants.EnvKeyOrganizationName].(string)
	organizationLogo := store[constants.EnvKeyOrganizationLogo].(string)

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

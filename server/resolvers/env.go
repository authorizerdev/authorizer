package resolvers

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// EnvResolver is a resolver for config query
// This is admin only query
func EnvResolver(ctx context.Context) (*model.Env, error) {
	var res *model.Env

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin.")
		return res, fmt.Errorf("unauthorized")
	}

	// get clone of store
	store, err := memorystore.Provider.GetEnvStore()
	if err != nil {
		log.Debug("Failed to get env store: ", err)
		return res, err
	}
	accessTokenExpiryTime := store[constants.EnvKeyAccessTokenExpiryTime].(string)
	adminSecret := store[constants.EnvKeyAdminSecret].(string)
	clientID := store[constants.EnvKeyClientID].(string)
	clientSecret := store[constants.EnvKeyClientSecret].(string)
	databaseURL := store[constants.EnvKeyDatabaseURL].(string)
	databaseName := store[constants.EnvKeyDatabaseName].(string)
	databaseType := store[constants.EnvKeyDatabaseType].(string)
	databaseUsername := store[constants.EnvKeyDatabaseUsername].(string)
	databasePassword := store[constants.EnvKeyDatabasePassword].(string)
	databaseHost := store[constants.EnvKeyDatabaseHost].(string)
	databasePort := store[constants.EnvKeyDatabasePort].(string)
	customAccessTokenScript := store[constants.EnvKeyCustomAccessTokenScript].(string)
	smtpHost := store[constants.EnvKeySmtpHost].(string)
	smtpPort := store[constants.EnvKeySmtpPort].(string)
	smtpUsername := store[constants.EnvKeySmtpUsername].(string)
	smtpPassword := store[constants.EnvKeySmtpPassword].(string)
	senderEmail := store[constants.EnvKeySenderEmail].(string)
	jwtType := store[constants.EnvKeyJwtType].(string)
	jwtSecret := store[constants.EnvKeyJwtSecret].(string)
	jwtRoleClaim := store[constants.EnvKeyJwtRoleClaim].(string)
	jwtPublicKey := store[constants.EnvKeyJwtPublicKey].(string)
	jwtPrivateKey := store[constants.EnvKeyJwtPrivateKey].(string)
	appURL := store[constants.EnvKeyAppURL].(string)
	redisURL := store[constants.EnvKeyRedisURL].(string)
	resetPasswordURL := store[constants.EnvKeyResetPasswordURL].(string)
	googleClientID := store[constants.EnvKeyGoogleClientID].(string)
	googleClientSecret := store[constants.EnvKeyGoogleClientSecret].(string)
	facebookClientID := store[constants.EnvKeyFacebookClientID].(string)
	facebookClientSecret := store[constants.EnvKeyFacebookClientSecret].(string)
	githubClientID := store[constants.EnvKeyGithubClientID].(string)
	githubClientSecret := store[constants.EnvKeyGithubClientSecret].(string)
	organizationName := store[constants.EnvKeyOrganizationName].(string)
	organizationLogo := store[constants.EnvKeyOrganizationLogo].(string)

	// string slice vars
	allowedOrigins := utils.ConvertInterfaceToStringSlice(store[constants.EnvKeyAllowedOrigins])
	roles := utils.ConvertInterfaceToStringSlice(store[constants.EnvKeyRoles])
	defaultRoles := utils.ConvertInterfaceToStringSlice(store[constants.EnvKeyDefaultRoles])
	protectedRoles := utils.ConvertInterfaceToStringSlice(store[constants.EnvKeyProtectedRoles])

	// bool vars
	disableEmailVerification := store[constants.EnvKeyDisableEmailVerification].(bool)
	disableBasicAuthentication := store[constants.EnvKeyDisableBasicAuthentication].(bool)
	disableMagicLinkLogin := store[constants.EnvKeyDisableMagicLinkLogin].(bool)
	disableLoginPage := store[constants.EnvKeyDisableLoginPage].(bool)
	disableSignUp := store[constants.EnvKeyDisableSignUp].(bool)

	if accessTokenExpiryTime == "" {
		accessTokenExpiryTime = "30m"
	}

	res = &model.Env{
		AccessTokenExpiryTime:      &accessTokenExpiryTime,
		AdminSecret:                &adminSecret,
		DatabaseName:               databaseName,
		DatabaseURL:                databaseURL,
		DatabaseType:               databaseType,
		DatabaseUsername:           databaseUsername,
		DatabasePassword:           databasePassword,
		DatabaseHost:               databaseHost,
		DatabasePort:               databasePort,
		ClientID:                   clientID,
		ClientSecret:               clientSecret,
		CustomAccessTokenScript:    &customAccessTokenScript,
		SMTPHost:                   &smtpHost,
		SMTPPort:                   &smtpPort,
		SMTPPassword:               &smtpPassword,
		SMTPUsername:               &smtpUsername,
		SenderEmail:                &senderEmail,
		JwtType:                    &jwtType,
		JwtSecret:                  &jwtSecret,
		JwtPrivateKey:              &jwtPrivateKey,
		JwtPublicKey:               &jwtPublicKey,
		JwtRoleClaim:               &jwtRoleClaim,
		AllowedOrigins:             allowedOrigins,
		AppURL:                     &appURL,
		RedisURL:                   &redisURL,
		ResetPasswordURL:           &resetPasswordURL,
		DisableEmailVerification:   &disableEmailVerification,
		DisableBasicAuthentication: &disableBasicAuthentication,
		DisableMagicLinkLogin:      &disableMagicLinkLogin,
		DisableLoginPage:           &disableLoginPage,
		DisableSignUp:              &disableSignUp,
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

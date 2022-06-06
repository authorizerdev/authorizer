package resolvers

import (
	"context"
	"fmt"
	"strings"

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
	res := &model.Env{}

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

	if val, ok := store[constants.EnvKeyAccessTokenExpiryTime]; ok {
		res.AccessTokenExpiryTime = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyAdminSecret]; ok {
		res.AdminSecret = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyClientID]; ok {
		res.ClientID = val.(string)
	}
	if val, ok := store[constants.EnvKeyClientSecret]; ok {
		res.ClientSecret = val.(string)
	}
	if val, ok := store[constants.EnvKeyDatabaseURL]; ok {
		res.DatabaseURL = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabaseName]; ok {
		res.DatabaseName = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabaseType]; ok {
		res.DatabaseType = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabaseUsername]; ok {
		res.DatabaseUsername = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabasePassword]; ok {
		res.DatabasePassword = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabaseHost]; ok {
		res.DatabaseHost = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabasePort]; ok {
		res.DatabasePort = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyCustomAccessTokenScript]; ok {
		res.CustomAccessTokenScript = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeySmtpHost]; ok {
		res.SMTPHost = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeySmtpPort]; ok {
		res.SMTPPort = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeySmtpUsername]; ok {
		res.SMTPUsername = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeySmtpPassword]; ok {
		res.SMTPPassword = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeySenderEmail]; ok {
		res.SenderEmail = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyJwtType]; ok {
		res.JwtType = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyJwtSecret]; ok {
		res.JwtSecret = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyJwtRoleClaim]; ok {
		res.JwtRoleClaim = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyJwtPublicKey]; ok {
		res.JwtPublicKey = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyJwtPrivateKey]; ok {
		res.JwtPrivateKey = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyAppURL]; ok {
		res.AppURL = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyRedisURL]; ok {
		res.RedisURL = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyResetPasswordURL]; ok {
		res.ResetPasswordURL = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyGoogleClientID]; ok {
		res.GoogleClientID = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyGoogleClientSecret]; ok {
		res.GoogleClientSecret = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyFacebookClientID]; ok {
		res.FacebookClientID = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyFacebookClientSecret]; ok {
		res.FacebookClientSecret = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyGithubClientID]; ok {
		res.GithubClientID = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyGithubClientSecret]; ok {
		res.GithubClientSecret = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyLinkedInClientID]; ok {
		res.LinkedinClientID = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyLinkedInClientSecret]; ok {
		res.LinkedinClientSecret = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyOrganizationName]; ok {
		res.OrganizationName = utils.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyOrganizationLogo]; ok {
		res.OrganizationLogo = utils.NewStringRef(val.(string))
	}

	// string slice vars
	res.AllowedOrigins = strings.Split(store[constants.EnvKeyAllowedOrigins].(string), ",")
	res.Roles = strings.Split(store[constants.EnvKeyRoles].(string), ",")
	res.DefaultRoles = strings.Split(store[constants.EnvKeyDefaultRoles].(string), ",")
	res.ProtectedRoles = strings.Split(store[constants.EnvKeyProtectedRoles].(string), ",")

	// bool vars
	res.DisableEmailVerification = store[constants.EnvKeyDisableEmailVerification].(bool)
	res.DisableBasicAuthentication = store[constants.EnvKeyDisableBasicAuthentication].(bool)
	res.DisableMagicLinkLogin = store[constants.EnvKeyDisableMagicLinkLogin].(bool)
	res.DisableLoginPage = store[constants.EnvKeyDisableLoginPage].(bool)
	res.DisableSignUp = store[constants.EnvKeyDisableSignUp].(bool)

	return res, nil
}

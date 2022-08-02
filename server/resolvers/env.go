package resolvers

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
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
		res.AccessTokenExpiryTime = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyAdminSecret]; ok {
		res.AdminSecret = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyClientID]; ok {
		res.ClientID = val.(string)
	}
	if val, ok := store[constants.EnvKeyClientSecret]; ok {
		res.ClientSecret = val.(string)
	}
	if val, ok := store[constants.EnvKeyDatabaseURL]; ok {
		res.DatabaseURL = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabaseName]; ok {
		res.DatabaseName = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabaseType]; ok {
		res.DatabaseType = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabaseUsername]; ok {
		res.DatabaseUsername = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabasePassword]; ok {
		res.DatabasePassword = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabaseHost]; ok {
		res.DatabaseHost = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyDatabasePort]; ok {
		res.DatabasePort = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyCustomAccessTokenScript]; ok {
		res.CustomAccessTokenScript = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeySmtpHost]; ok {
		res.SMTPHost = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeySmtpPort]; ok {
		res.SMTPPort = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeySmtpUsername]; ok {
		res.SMTPUsername = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeySmtpPassword]; ok {
		res.SMTPPassword = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeySenderEmail]; ok {
		res.SenderEmail = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyJwtType]; ok {
		res.JwtType = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyJwtSecret]; ok {
		res.JwtSecret = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyJwtRoleClaim]; ok {
		res.JwtRoleClaim = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyJwtPublicKey]; ok {
		res.JwtPublicKey = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyJwtPrivateKey]; ok {
		res.JwtPrivateKey = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyAppURL]; ok {
		res.AppURL = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyRedisURL]; ok {
		res.RedisURL = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyResetPasswordURL]; ok {
		res.ResetPasswordURL = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyGoogleClientID]; ok {
		res.GoogleClientID = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyGoogleClientSecret]; ok {
		res.GoogleClientSecret = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyFacebookClientID]; ok {
		res.FacebookClientID = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyFacebookClientSecret]; ok {
		res.FacebookClientSecret = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyGithubClientID]; ok {
		res.GithubClientID = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyGithubClientSecret]; ok {
		res.GithubClientSecret = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyLinkedInClientID]; ok {
		res.LinkedinClientID = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyLinkedInClientSecret]; ok {
		res.LinkedinClientSecret = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyAppleClientID]; ok {
		res.AppleClientID = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyAppleClientSecret]; ok {
		res.AppleClientSecret = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyOrganizationName]; ok {
		res.OrganizationName = refs.NewStringRef(val.(string))
	}
	if val, ok := store[constants.EnvKeyOrganizationLogo]; ok {
		res.OrganizationLogo = refs.NewStringRef(val.(string))
	}

	// string slice vars
	res.AllowedOrigins = strings.Split(store[constants.EnvKeyAllowedOrigins].(string), ",")
	res.Roles = strings.Split(store[constants.EnvKeyRoles].(string), ",")
	res.DefaultRoles = strings.Split(store[constants.EnvKeyDefaultRoles].(string), ",")
	// since protected role is optional default split gives array with empty string
	protectedRoles := strings.Split(store[constants.EnvKeyProtectedRoles].(string), ",")
	res.ProtectedRoles = []string{}
	for _, role := range protectedRoles {
		if strings.Trim(role, " ") != "" {
			res.ProtectedRoles = append(res.ProtectedRoles, strings.Trim(role, " "))
		}
	}

	// bool vars
	res.DisableEmailVerification = store[constants.EnvKeyDisableEmailVerification].(bool)
	res.DisableBasicAuthentication = store[constants.EnvKeyDisableBasicAuthentication].(bool)
	res.DisableMagicLinkLogin = store[constants.EnvKeyDisableMagicLinkLogin].(bool)
	res.DisableLoginPage = store[constants.EnvKeyDisableLoginPage].(bool)
	res.DisableSignUp = store[constants.EnvKeyDisableSignUp].(bool)
	res.DisableStrongPassword = store[constants.EnvKeyDisableStrongPassword].(bool)
	res.EnforceMultiFactorAuthentication = store[constants.EnvKeyEnforceMultiFactorAuthentication].(bool)

	return res, nil
}

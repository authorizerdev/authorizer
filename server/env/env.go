package env

import (
	"log"
	"os"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// TODO move this to env store
var (
	// ARG_DB_URL is the cli arg variable for the database url
	ARG_DB_URL *string
	// ARG_DB_TYPE is the cli arg variable for the database type
	ARG_DB_TYPE *string
	// ARG_ENV_FILE is the cli arg variable for the env file
	ARG_ENV_FILE *string
)

// InitEnv to initialize EnvData and through error if required env are not present
func InitEnv() {
	// get clone of current store
	envData := envstore.EnvInMemoryStoreObj.GetEnvStoreClone()

	if envData[constants.EnvKeyEnv] == nil || envData[constants.EnvKeyEnv] == "" {
		envData[constants.EnvKeyEnv] = os.Getenv("ENV")
		if envData[constants.EnvKeyEnv] == "" {
			envData[constants.EnvKeyEnv] = "production"
		}

		if envData[constants.EnvKeyEnv] == "production" {
			envData[constants.EnvKeyIsProd] = true
			os.Setenv("GIN_MODE", "release")
		} else {
			envData[constants.EnvKeyIsProd] = false
		}
	}

	// set authorizer url to empty string so that fresh url is obtained with every server start
	envData[constants.EnvKeyAuthorizerURL] = ""
	if envData[constants.EnvKeyAppURL] == nil || envData[constants.EnvKeyAppURL] == "" {
		envData[constants.EnvKeyAppURL] = os.Getenv(constants.EnvKeyAppURL)
	}

	if envData[constants.EnvKeyEnvPath] == nil || envData[constants.EnvKeyEnvPath].(string) == "" {
		envData[constants.EnvKeyEnvPath] = `.env`
	}

	if ARG_ENV_FILE != nil && *ARG_ENV_FILE != "" {
		envData[constants.EnvKeyEnvPath] = *ARG_ENV_FILE
	}

	err := godotenv.Load(envData[constants.EnvKeyEnvPath].(string))
	if err != nil {
		log.Printf("error loading %s file", envData[constants.EnvKeyEnvPath])
	}

	if envData[constants.EnvKeyPort] == nil || envData[constants.EnvKeyPort].(string) == "" {
		envData[constants.EnvKeyPort] = os.Getenv("PORT")
		if envData[constants.EnvKeyPort].(string) == "" {
			envData[constants.EnvKeyPort] = "8080"
		}
	}

	if envData[constants.EnvKeyAdminSecret] == nil || envData[constants.EnvKeyAdminSecret].(string) == "" {
		envData[constants.EnvKeyAdminSecret] = os.Getenv("ADMIN_SECRET")
	}

	if envData[constants.EnvKeyDatabaseType] == nil || envData[constants.EnvKeyDatabaseType].(string) == "" {
		envData[constants.EnvKeyDatabaseType] = os.Getenv("DATABASE_TYPE")
		log.Println(envData[constants.EnvKeyDatabaseType].(string))

		if ARG_DB_TYPE != nil && *ARG_DB_TYPE != "" {
			envData[constants.EnvKeyDatabaseType] = *ARG_DB_TYPE
		}

		if envData[constants.EnvKeyDatabaseType].(string) == "" {
			panic("DATABASE_TYPE is required")
		}
	}

	if envData[constants.EnvKeyDatabaseURL] == nil || envData[constants.EnvKeyDatabaseURL].(string) == "" {
		envData[constants.EnvKeyDatabaseURL] = os.Getenv("DATABASE_URL")

		if ARG_DB_URL != nil && *ARG_DB_URL != "" {
			envData[constants.EnvKeyDatabaseURL] = *ARG_DB_URL
		}

		if envData[constants.EnvKeyDatabaseURL] == "" {
			panic("DATABASE_URL is required")
		}
	}

	if envData[constants.EnvKeyDatabaseName] == nil || envData[constants.EnvKeyDatabaseName].(string) == "" {
		envData[constants.EnvKeyDatabaseName] = os.Getenv("DATABASE_NAME")
		if envData[constants.EnvKeyDatabaseName].(string) == "" {
			envData[constants.EnvKeyDatabaseName] = "authorizer"
		}
	}

	if envData[constants.EnvKeySmtpHost] == nil || envData[constants.EnvKeySmtpHost].(string) == "" {
		envData[constants.EnvKeySmtpHost] = os.Getenv("SMTP_HOST")
	}

	if envData[constants.EnvKeySmtpPort] == nil || envData[constants.EnvKeySmtpPort].(string) == "" {
		envData[constants.EnvKeySmtpPort] = os.Getenv("SMTP_PORT")
	}

	if envData[constants.EnvKeySmtpUsername] == nil || envData[constants.EnvKeySmtpUsername].(string) == "" {
		envData[constants.EnvKeySmtpUsername] = os.Getenv("SMTP_USERNAME")
	}

	if envData[constants.EnvKeySmtpPassword] == nil || envData[constants.EnvKeySmtpPassword].(string) == "" {
		envData[constants.EnvKeySmtpPassword] = os.Getenv("SMTP_PASSWORD")
	}

	if envData[constants.EnvKeySenderEmail] == nil || envData[constants.EnvKeySenderEmail].(string) == "" {
		envData[constants.EnvKeySenderEmail] = os.Getenv("SENDER_EMAIL")
	}

	if envData[constants.EnvKeyJwtSecret] == nil || envData[constants.EnvKeyJwtSecret].(string) == "" {
		envData[constants.EnvKeyJwtSecret] = os.Getenv("JWT_SECRET")
		if envData[constants.EnvKeyJwtSecret].(string) == "" {
			envData[constants.EnvKeyJwtSecret] = uuid.New().String()
		}
	}

	if envData[constants.EnvKeyJwtType] == nil || envData[constants.EnvKeyJwtType].(string) == "" {
		envData[constants.EnvKeyJwtType] = os.Getenv("JWT_TYPE")
		if envData[constants.EnvKeyJwtType].(string) == "" {
			envData[constants.EnvKeyJwtType] = "HS256"
		}
	}

	if envData[constants.EnvKeyJwtRoleClaim] == nil || envData[constants.EnvKeyJwtRoleClaim].(string) == "" {
		envData[constants.EnvKeyJwtRoleClaim] = os.Getenv("JWT_ROLE_CLAIM")

		if envData[constants.EnvKeyJwtRoleClaim].(string) == "" {
			envData[constants.EnvKeyJwtRoleClaim] = "role"
		}
	}

	if envData[constants.EnvKeyRedisURL] == nil || envData[constants.EnvKeyRedisURL].(string) == "" {
		envData[constants.EnvKeyRedisURL] = os.Getenv("REDIS_URL")
	}

	if envData[constants.EnvKeyCookieName] == nil || envData[constants.EnvKeyCookieName].(string) == "" {
		envData[constants.EnvKeyCookieName] = os.Getenv("COOKIE_NAME")
		if envData[constants.EnvKeyCookieName].(string) == "" {
			envData[constants.EnvKeyCookieName] = "authorizer"
		}
	}

	if envData[constants.EnvKeyGoogleClientID] == nil || envData[constants.EnvKeyGoogleClientID].(string) == "" {
		envData[constants.EnvKeyGoogleClientID] = os.Getenv("GOOGLE_CLIENT_ID")
	}

	if envData[constants.EnvKeyGoogleClientSecret] == nil || envData[constants.EnvKeyGoogleClientSecret].(string) == "" {
		envData[constants.EnvKeyGoogleClientSecret] = os.Getenv("GOOGLE_CLIENT_SECRET")
	}

	if envData[constants.EnvKeyGithubClientID] == nil || envData[constants.EnvKeyGithubClientID].(string) == "" {
		envData[constants.EnvKeyGithubClientID] = os.Getenv("GITHUB_CLIENT_ID")
	}

	if envData[constants.EnvKeyGithubClientSecret] == nil || envData[constants.EnvKeyGithubClientSecret].(string) == "" {
		envData[constants.EnvKeyGithubClientSecret] = os.Getenv("GITHUB_CLIENT_SECRET")
	}

	if envData[constants.EnvKeyFacebookClientID] == nil || envData[constants.EnvKeyFacebookClientID].(string) == "" {
		envData[constants.EnvKeyFacebookClientID] = os.Getenv("FACEBOOK_CLIENT_ID")
	}

	if envData[constants.EnvKeyFacebookClientSecret] == nil || envData[constants.EnvKeyFacebookClientSecret].(string) == "" {
		envData[constants.EnvKeyFacebookClientSecret] = os.Getenv("FACEBOOK_CLIENT_SECRET")
	}

	if envData[constants.EnvKeyResetPasswordURL] == nil || envData[constants.EnvKeyResetPasswordURL].(string) == "" {
		envData[constants.EnvKeyResetPasswordURL] = strings.TrimPrefix(os.Getenv("RESET_PASSWORD_URL"), "/")
	}

	envData[constants.EnvKeyDisableBasicAuthentication] = os.Getenv("DISABLE_BASIC_AUTHENTICATION") == "true"
	envData[constants.EnvKeyDisableEmailVerification] = os.Getenv("DISABLE_EMAIL_VERIFICATION") == "true"
	envData[constants.EnvKeyDisableMagicLinkLogin] = os.Getenv("DISABLE_MAGIC_LINK_LOGIN") == "true"
	envData[constants.EnvKeyDisableLoginPage] = os.Getenv("DISABLE_LOGIN_PAGE") == "true"

	// no need to add nil check as its already done above
	if envData[constants.EnvKeySmtpHost].(string) == "" || envData[constants.EnvKeySmtpUsername].(string) == "" || envData[constants.EnvKeySmtpPassword].(string) == "" || envData[constants.EnvKeySenderEmail].(string) == "" {
		envData[constants.EnvKeyDisableEmailVerification] = true
		envData[constants.EnvKeyDisableMagicLinkLogin] = true
	}

	if envData[constants.EnvKeyDisableEmailVerification].(bool) {
		envData[constants.EnvKeyDisableMagicLinkLogin] = true
	}

	allowedOriginsSplit := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	allowedOrigins := []string{}
	hasWildCard := false

	for _, val := range allowedOriginsSplit {
		trimVal := strings.TrimSpace(val)
		if trimVal != "" {
			if trimVal != "*" {
				host, port := utils.GetHostParts(trimVal)
				allowedOrigins = append(allowedOrigins, host+":"+port)
			} else {
				hasWildCard = true
				allowedOrigins = append(allowedOrigins, trimVal)
				break
			}
		}
	}

	if len(allowedOrigins) > 1 && hasWildCard {
		allowedOrigins = []string{"*"}
	}

	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	envData[constants.EnvKeyAllowedOrigins] = allowedOrigins

	rolesEnv := strings.TrimSpace(os.Getenv("ROLES"))
	rolesSplit := strings.Split(rolesEnv, ",")
	roles := []string{}
	if len(rolesEnv) == 0 {
		roles = []string{"user"}
	}

	defaultRolesEnv := strings.TrimSpace(os.Getenv("DEFAULT_ROLES"))
	defaultRoleSplit := strings.Split(defaultRolesEnv, ",")
	defaultRoles := []string{}

	if len(defaultRolesEnv) == 0 {
		defaultRoles = []string{"user"}
	}

	protectedRolesEnv := strings.TrimSpace(os.Getenv("PROTECTED_ROLES"))
	protectedRolesSplit := strings.Split(protectedRolesEnv, ",")
	protectedRoles := []string{}

	if len(protectedRolesEnv) > 0 {
		for _, val := range protectedRolesSplit {
			trimVal := strings.TrimSpace(val)
			protectedRoles = append(protectedRoles, trimVal)
		}
	}

	for _, val := range rolesSplit {
		trimVal := strings.TrimSpace(val)
		if trimVal != "" {
			roles = append(roles, trimVal)
		}

		if utils.StringSliceContains(defaultRoleSplit, trimVal) {
			defaultRoles = append(defaultRoles, trimVal)
		}
	}

	if len(roles) > 0 && len(defaultRoles) == 0 && len(defaultRolesEnv) > 0 {
		panic(`Invalid DEFAULT_ROLE environment variable. It can be one from give ROLES environment variable value`)
	}

	envData[constants.EnvKeyRoles] = roles
	envData[constants.EnvKeyDefaultRoles] = defaultRoles
	envData[constants.EnvKeyProtectedRoles] = protectedRoles

	if os.Getenv("ORGANIZATION_NAME") != "" {
		envData[constants.EnvKeyOrganizationName] = os.Getenv("ORGANIZATION_NAME")
	}

	if os.Getenv("ORGANIZATION_LOGO") != "" {
		envData[constants.EnvKeyOrganizationLogo] = os.Getenv("ORGANIZATION_LOGO")
	}

	envstore.EnvInMemoryStoreObj.UpdateEnvStore(envData)
}

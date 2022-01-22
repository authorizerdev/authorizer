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

// InitEnv to initialize EnvData and through error if required env are not present
func InitEnv() {
	// get clone of current store
	envData := envstore.EnvInMemoryStoreObj.GetEnvStoreClone()

	if envData.StringEnv[constants.EnvKeyEnv] == "" {
		envData.StringEnv[constants.EnvKeyEnv] = os.Getenv("ENV")
		if envData.StringEnv[constants.EnvKeyEnv] == "" {
			envData.StringEnv[constants.EnvKeyEnv] = "production"
		}

		if envData.StringEnv[constants.EnvKeyEnv] == "production" {
			envData.BoolEnv[constants.EnvKeyIsProd] = true
			os.Setenv("GIN_MODE", "release")
		} else {
			envData.BoolEnv[constants.EnvKeyIsProd] = false
		}
	}

	// set authorizer url to empty string so that fresh url is obtained with every server start
	envData.StringEnv[constants.EnvKeyAuthorizerURL] = ""
	if envData.StringEnv[constants.EnvKeyAppURL] == "" {
		envData.StringEnv[constants.EnvKeyAppURL] = os.Getenv(constants.EnvKeyAppURL)
	}

	if envData.StringEnv[constants.EnvKeyEnvPath] == "" {
		envData.StringEnv[constants.EnvKeyEnvPath] = `.env`
	}

	if envstore.ARG_ENV_FILE != nil && *envstore.ARG_ENV_FILE != "" {
		envData.StringEnv[constants.EnvKeyEnvPath] = *envstore.ARG_ENV_FILE
	}

	err := godotenv.Load(envData.StringEnv[constants.EnvKeyEnvPath])
	if err != nil {
		log.Printf("error loading %s file", envData.StringEnv[constants.EnvKeyEnvPath])
	}

	if envData.StringEnv[constants.EnvKeyPort] == "" {
		envData.StringEnv[constants.EnvKeyPort] = os.Getenv("PORT")
		if envData.StringEnv[constants.EnvKeyPort] == "" {
			envData.StringEnv[constants.EnvKeyPort] = "8080"
		}
	}

	if envData.StringEnv[constants.EnvKeyAdminSecret] == "" {
		envData.StringEnv[constants.EnvKeyAdminSecret] = os.Getenv("ADMIN_SECRET")
	}

	if envData.StringEnv[constants.EnvKeyDatabaseType] == "" {
		envData.StringEnv[constants.EnvKeyDatabaseType] = os.Getenv("DATABASE_TYPE")

		if envstore.ARG_DB_TYPE != nil && *envstore.ARG_DB_TYPE != "" {
			envData.StringEnv[constants.EnvKeyDatabaseType] = *envstore.ARG_DB_TYPE
		}

		if envData.StringEnv[constants.EnvKeyDatabaseType] == "" {
			panic("DATABASE_TYPE is required")
		}
	}

	if envData.StringEnv[constants.EnvKeyDatabaseURL] == "" {
		envData.StringEnv[constants.EnvKeyDatabaseURL] = os.Getenv("DATABASE_URL")

		if envstore.ARG_DB_URL != nil && *envstore.ARG_DB_URL != "" {
			envData.StringEnv[constants.EnvKeyDatabaseURL] = *envstore.ARG_DB_URL
		}

		if envData.StringEnv[constants.EnvKeyDatabaseURL] == "" {
			panic("DATABASE_URL is required")
		}
	}

	if envData.StringEnv[constants.EnvKeyDatabaseName] == "" {
		envData.StringEnv[constants.EnvKeyDatabaseName] = os.Getenv("DATABASE_NAME")
		if envData.StringEnv[constants.EnvKeyDatabaseName] == "" {
			envData.StringEnv[constants.EnvKeyDatabaseName] = "authorizer"
		}
	}

	if envData.StringEnv[constants.EnvKeySmtpHost] == "" {
		envData.StringEnv[constants.EnvKeySmtpHost] = os.Getenv("SMTP_HOST")
	}

	if envData.StringEnv[constants.EnvKeySmtpPort] == "" {
		envData.StringEnv[constants.EnvKeySmtpPort] = os.Getenv("SMTP_PORT")
	}

	if envData.StringEnv[constants.EnvKeySmtpUsername] == "" {
		envData.StringEnv[constants.EnvKeySmtpUsername] = os.Getenv("SMTP_USERNAME")
	}

	if envData.StringEnv[constants.EnvKeySmtpPassword] == "" {
		envData.StringEnv[constants.EnvKeySmtpPassword] = os.Getenv("SMTP_PASSWORD")
	}

	if envData.StringEnv[constants.EnvKeySenderEmail] == "" {
		envData.StringEnv[constants.EnvKeySenderEmail] = os.Getenv("SENDER_EMAIL")
	}

	if envData.StringEnv[constants.EnvKeyJwtSecret] == "" {
		envData.StringEnv[constants.EnvKeyJwtSecret] = os.Getenv("JWT_SECRET")
		if envData.StringEnv[constants.EnvKeyJwtSecret] == "" {
			envData.StringEnv[constants.EnvKeyJwtSecret] = uuid.New().String()
		}
	}

	if envData.StringEnv[constants.EnvKeyJwtType] == "" {
		envData.StringEnv[constants.EnvKeyJwtType] = os.Getenv("JWT_TYPE")
		if envData.StringEnv[constants.EnvKeyJwtType] == "" {
			envData.StringEnv[constants.EnvKeyJwtType] = "HS256"
		}
	}

	if envData.StringEnv[constants.EnvKeyJwtRoleClaim] == "" {
		envData.StringEnv[constants.EnvKeyJwtRoleClaim] = os.Getenv("JWT_ROLE_CLAIM")

		if envData.StringEnv[constants.EnvKeyJwtRoleClaim] == "" {
			envData.StringEnv[constants.EnvKeyJwtRoleClaim] = "role"
		}
	}

	if envData.StringEnv[constants.EnvKeyRedisURL] == "" {
		envData.StringEnv[constants.EnvKeyRedisURL] = os.Getenv("REDIS_URL")
	}

	if envData.StringEnv[constants.EnvKeyCookieName] == "" {
		envData.StringEnv[constants.EnvKeyCookieName] = os.Getenv("COOKIE_NAME")
		if envData.StringEnv[constants.EnvKeyCookieName] == "" {
			envData.StringEnv[constants.EnvKeyCookieName] = "authorizer"
		}
	}

	if envData.StringEnv[constants.EnvKeyGoogleClientID] == "" {
		envData.StringEnv[constants.EnvKeyGoogleClientID] = os.Getenv("GOOGLE_CLIENT_ID")
	}

	if envData.StringEnv[constants.EnvKeyGoogleClientSecret] == "" {
		envData.StringEnv[constants.EnvKeyGoogleClientSecret] = os.Getenv("GOOGLE_CLIENT_SECRET")
	}

	if envData.StringEnv[constants.EnvKeyGithubClientID] == "" {
		envData.StringEnv[constants.EnvKeyGithubClientID] = os.Getenv("GITHUB_CLIENT_ID")
	}

	if envData.StringEnv[constants.EnvKeyGithubClientSecret] == "" {
		envData.StringEnv[constants.EnvKeyGithubClientSecret] = os.Getenv("GITHUB_CLIENT_SECRET")
	}

	if envData.StringEnv[constants.EnvKeyFacebookClientID] == "" {
		envData.StringEnv[constants.EnvKeyFacebookClientID] = os.Getenv("FACEBOOK_CLIENT_ID")
	}

	if envData.StringEnv[constants.EnvKeyFacebookClientSecret] == "" {
		envData.StringEnv[constants.EnvKeyFacebookClientSecret] = os.Getenv("FACEBOOK_CLIENT_SECRET")
	}

	if envData.StringEnv[constants.EnvKeyResetPasswordURL] == "" {
		envData.StringEnv[constants.EnvKeyResetPasswordURL] = strings.TrimPrefix(os.Getenv("RESET_PASSWORD_URL"), "/")
	}

	envData.BoolEnv[constants.EnvKeyDisableBasicAuthentication] = os.Getenv("DISABLE_BASIC_AUTHENTICATION") == "true"
	envData.BoolEnv[constants.EnvKeyDisableEmailVerification] = os.Getenv("DISABLE_EMAIL_VERIFICATION") == "true"
	envData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = os.Getenv("DISABLE_MAGIC_LINK_LOGIN") == "true"
	envData.BoolEnv[constants.EnvKeyDisableLoginPage] = os.Getenv("DISABLE_LOGIN_PAGE") == "true"

	// no need to add nil check as its already done above
	if envData.StringEnv[constants.EnvKeySmtpHost] == "" || envData.StringEnv[constants.EnvKeySmtpUsername] == "" || envData.StringEnv[constants.EnvKeySmtpPassword] == "" || envData.StringEnv[constants.EnvKeySenderEmail] == "" && envData.StringEnv[constants.EnvKeySmtpPort] == "" {
		envData.BoolEnv[constants.EnvKeyDisableEmailVerification] = true
		envData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = true
	}

	if envData.BoolEnv[constants.EnvKeyDisableEmailVerification] {
		envData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = true
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

	envData.SliceEnv[constants.EnvKeyAllowedOrigins] = allowedOrigins

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

	envData.SliceEnv[constants.EnvKeyRoles] = roles
	envData.SliceEnv[constants.EnvKeyDefaultRoles] = defaultRoles
	envData.SliceEnv[constants.EnvKeyProtectedRoles] = protectedRoles

	if os.Getenv("ORGANIZATION_NAME") != "" {
		envData.StringEnv[constants.EnvKeyOrganizationName] = os.Getenv("ORGANIZATION_NAME")
	}

	if os.Getenv("ORGANIZATION_LOGO") != "" {
		envData.StringEnv[constants.EnvKeyOrganizationLogo] = os.Getenv("ORGANIZATION_LOGO")
	}

	envstore.EnvInMemoryStoreObj.UpdateEnvStore(envData)
}

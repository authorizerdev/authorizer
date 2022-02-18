package env

import (
	"log"
	"os"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// InitEnv to initialize EnvData and through error if required env are not present
func InitEnv() {
	// get clone of current store
	envData := envstore.EnvInMemoryStoreObj.GetEnvStoreClone()

	if envData.StringEnv[constants.EnvKeyEnv] == "" {
		envData.StringEnv[constants.EnvKeyEnv] = os.Getenv(constants.EnvKeyEnv)
		if envData.StringEnv[constants.EnvKeyEnv] == "" {
			envData.StringEnv[constants.EnvKeyEnv] = "production"
		}

		if envData.StringEnv[constants.EnvKeyEnv] == "production" {
			envData.BoolEnv[constants.EnvKeyIsProd] = true
			gin.SetMode(gin.ReleaseMode)
		} else {
			envData.BoolEnv[constants.EnvKeyIsProd] = false
		}
	}

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
		log.Printf("using OS env instead of %s file", envData.StringEnv[constants.EnvKeyEnvPath])
	}

	if envData.StringEnv[constants.EnvKeyPort] == "" {
		envData.StringEnv[constants.EnvKeyPort] = os.Getenv(constants.EnvKeyPort)
		if envData.StringEnv[constants.EnvKeyPort] == "" {
			envData.StringEnv[constants.EnvKeyPort] = "8080"
		}
	}

	if envData.StringEnv[constants.EnvKeyAdminSecret] == "" {
		envData.StringEnv[constants.EnvKeyAdminSecret] = os.Getenv(constants.EnvKeyAdminSecret)
	}

	if envData.StringEnv[constants.EnvKeyDatabaseType] == "" {
		envData.StringEnv[constants.EnvKeyDatabaseType] = os.Getenv(constants.EnvKeyDatabaseType)

		if envstore.ARG_DB_TYPE != nil && *envstore.ARG_DB_TYPE != "" {
			envData.StringEnv[constants.EnvKeyDatabaseType] = *envstore.ARG_DB_TYPE
		}

		if envData.StringEnv[constants.EnvKeyDatabaseType] == "" {
			panic("DATABASE_TYPE is required")
		}
	}

	if envData.StringEnv[constants.EnvKeyDatabaseURL] == "" {
		envData.StringEnv[constants.EnvKeyDatabaseURL] = os.Getenv(constants.EnvKeyDatabaseURL)

		if envstore.ARG_DB_URL != nil && *envstore.ARG_DB_URL != "" {
			envData.StringEnv[constants.EnvKeyDatabaseURL] = *envstore.ARG_DB_URL
		}

		if envData.StringEnv[constants.EnvKeyDatabaseURL] == "" {
			panic("DATABASE_URL is required")
		}
	}

	if envData.StringEnv[constants.EnvKeyDatabaseName] == "" {
		envData.StringEnv[constants.EnvKeyDatabaseName] = os.Getenv(constants.EnvKeyDatabaseName)
		if envData.StringEnv[constants.EnvKeyDatabaseName] == "" {
			envData.StringEnv[constants.EnvKeyDatabaseName] = "authorizer"
		}
	}

	if envData.StringEnv[constants.EnvKeySmtpHost] == "" {
		envData.StringEnv[constants.EnvKeySmtpHost] = os.Getenv(constants.EnvKeySmtpHost)
	}

	if envData.StringEnv[constants.EnvKeySmtpPort] == "" {
		envData.StringEnv[constants.EnvKeySmtpPort] = os.Getenv(constants.EnvKeySmtpPort)
	}

	if envData.StringEnv[constants.EnvKeySmtpUsername] == "" {
		envData.StringEnv[constants.EnvKeySmtpUsername] = os.Getenv(constants.EnvKeySmtpUsername)
	}

	if envData.StringEnv[constants.EnvKeySmtpPassword] == "" {
		envData.StringEnv[constants.EnvKeySmtpPassword] = os.Getenv(constants.EnvKeySmtpPassword)
	}

	if envData.StringEnv[constants.EnvKeySenderEmail] == "" {
		envData.StringEnv[constants.EnvKeySenderEmail] = os.Getenv(constants.EnvKeySenderEmail)
	}

	if envData.StringEnv[constants.EnvKeyJwtSecret] == "" {
		envData.StringEnv[constants.EnvKeyJwtSecret] = os.Getenv(constants.EnvKeyJwtSecret)
		if envData.StringEnv[constants.EnvKeyJwtSecret] == "" {
			envData.StringEnv[constants.EnvKeyJwtSecret] = uuid.New().String()
		}
	}

	if envData.StringEnv[constants.EnvKeyCustomAccessTokenScript] == "" {
		envData.StringEnv[constants.EnvKeyCustomAccessTokenScript] = os.Getenv(constants.EnvKeyCustomAccessTokenScript)
	}

	if envData.StringEnv[constants.EnvKeyJwtPrivateKey] == "" {
		envData.StringEnv[constants.EnvKeyJwtPrivateKey] = os.Getenv(constants.EnvKeyJwtPrivateKey)
	}

	if envData.StringEnv[constants.EnvKeyJwtPublicKey] == "" {
		envData.StringEnv[constants.EnvKeyJwtPublicKey] = os.Getenv(constants.EnvKeyJwtPublicKey)
	}

	if envData.StringEnv[constants.EnvKeyJwtType] == "" {
		envData.StringEnv[constants.EnvKeyJwtType] = os.Getenv(constants.EnvKeyJwtType)
		if envData.StringEnv[constants.EnvKeyJwtType] == "" {
			envData.StringEnv[constants.EnvKeyJwtType] = "HS256"
		}
	}

	if envData.StringEnv[constants.EnvKeyJwtRoleClaim] == "" {
		envData.StringEnv[constants.EnvKeyJwtRoleClaim] = os.Getenv(constants.EnvKeyJwtRoleClaim)

		if envData.StringEnv[constants.EnvKeyJwtRoleClaim] == "" {
			envData.StringEnv[constants.EnvKeyJwtRoleClaim] = "role"
		}
	}

	if envData.StringEnv[constants.EnvKeyRedisURL] == "" {
		envData.StringEnv[constants.EnvKeyRedisURL] = os.Getenv(constants.EnvKeyRedisURL)
	}

	if envData.StringEnv[constants.EnvKeyCookieName] == "" {
		envData.StringEnv[constants.EnvKeyCookieName] = os.Getenv(constants.EnvKeyCookieName)
		if envData.StringEnv[constants.EnvKeyCookieName] == "" {
			envData.StringEnv[constants.EnvKeyCookieName] = "authorizer"
		}
	}

	if envData.StringEnv[constants.EnvKeyGoogleClientID] == "" {
		envData.StringEnv[constants.EnvKeyGoogleClientID] = os.Getenv(constants.EnvKeyGoogleClientID)
	}

	if envData.StringEnv[constants.EnvKeyGoogleClientSecret] == "" {
		envData.StringEnv[constants.EnvKeyGoogleClientSecret] = os.Getenv(constants.EnvKeyGoogleClientSecret)
	}

	if envData.StringEnv[constants.EnvKeyGithubClientID] == "" {
		envData.StringEnv[constants.EnvKeyGithubClientID] = os.Getenv(constants.EnvKeyGithubClientID)
	}

	if envData.StringEnv[constants.EnvKeyGithubClientSecret] == "" {
		envData.StringEnv[constants.EnvKeyGithubClientSecret] = os.Getenv(constants.EnvKeyGithubClientSecret)
	}

	if envData.StringEnv[constants.EnvKeyFacebookClientID] == "" {
		envData.StringEnv[constants.EnvKeyFacebookClientID] = os.Getenv(constants.EnvKeyFacebookClientID)
	}

	if envData.StringEnv[constants.EnvKeyFacebookClientSecret] == "" {
		envData.StringEnv[constants.EnvKeyFacebookClientSecret] = os.Getenv(constants.EnvKeyFacebookClientSecret)
	}

	if envData.StringEnv[constants.EnvKeyResetPasswordURL] == "" {
		envData.StringEnv[constants.EnvKeyResetPasswordURL] = strings.TrimPrefix(os.Getenv(constants.EnvKeyResetPasswordURL), "/")
	}

	envData.BoolEnv[constants.EnvKeyDisableBasicAuthentication] = os.Getenv(constants.EnvKeyDisableBasicAuthentication) == "true"
	envData.BoolEnv[constants.EnvKeyDisableEmailVerification] = os.Getenv(constants.EnvKeyDisableEmailVerification) == "true"
	envData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = os.Getenv(constants.EnvKeyDisableMagicLinkLogin) == "true"
	envData.BoolEnv[constants.EnvKeyDisableLoginPage] = os.Getenv(constants.EnvKeyDisableLoginPage) == "true"

	// no need to add nil check as its already done above
	if envData.StringEnv[constants.EnvKeySmtpHost] == "" || envData.StringEnv[constants.EnvKeySmtpUsername] == "" || envData.StringEnv[constants.EnvKeySmtpPassword] == "" || envData.StringEnv[constants.EnvKeySenderEmail] == "" && envData.StringEnv[constants.EnvKeySmtpPort] == "" {
		envData.BoolEnv[constants.EnvKeyDisableEmailVerification] = true
		envData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = true
	}

	if envData.BoolEnv[constants.EnvKeyDisableEmailVerification] {
		envData.BoolEnv[constants.EnvKeyDisableMagicLinkLogin] = true
	}

	allowedOriginsSplit := strings.Split(os.Getenv(constants.EnvKeyAllowedOrigins), ",")
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

	rolesEnv := strings.TrimSpace(os.Getenv(constants.EnvKeyRoles))
	rolesSplit := strings.Split(rolesEnv, ",")
	roles := []string{}
	if len(rolesEnv) == 0 {
		roles = []string{"user"}
	}

	defaultRolesEnv := strings.TrimSpace(os.Getenv(constants.EnvKeyDefaultRoles))
	defaultRoleSplit := strings.Split(defaultRolesEnv, ",")
	defaultRoles := []string{}

	if len(defaultRolesEnv) == 0 {
		defaultRoles = []string{"user"}
	}

	protectedRolesEnv := strings.TrimSpace(os.Getenv(constants.EnvKeyProtectedRoles))
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
			if utils.StringSliceContains(defaultRoleSplit, trimVal) {
				defaultRoles = append(defaultRoles, trimVal)
			}
		}
	}

	if len(roles) > 0 && len(defaultRoles) == 0 && len(defaultRolesEnv) > 0 {
		panic(`Invalid DEFAULT_ROLE environment variable. It can be one from give ROLES environment variable value`)
	}

	envData.SliceEnv[constants.EnvKeyRoles] = roles
	envData.SliceEnv[constants.EnvKeyDefaultRoles] = defaultRoles
	envData.SliceEnv[constants.EnvKeyProtectedRoles] = protectedRoles

	if os.Getenv(constants.EnvKeyOrganizationName) != "" {
		envData.StringEnv[constants.EnvKeyOrganizationName] = os.Getenv(constants.EnvKeyOrganizationName)
	}

	if os.Getenv(constants.EnvKeyOrganizationLogo) != "" {
		envData.StringEnv[constants.EnvKeyOrganizationLogo] = os.Getenv(constants.EnvKeyOrganizationLogo)
	}

	envstore.EnvInMemoryStoreObj.UpdateEnvStore(envData)
}

package env

import (
	"log"
	"os"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// build variables
var (
	ARG_DB_URL         *string
	ARG_DB_TYPE        *string
	ARG_AUTHORIZER_URL *string
	ARG_ENV_FILE       *string
)

// InitEnv -> to initialize env and through error if required env are not present
func InitEnv() {
	if constants.EnvData.ENV_PATH == "" {
		constants.EnvData.ENV_PATH = `.env`
	}

	if ARG_ENV_FILE != nil && *ARG_ENV_FILE != "" {
		constants.EnvData.ENV_PATH = *ARG_ENV_FILE
	}

	err := godotenv.Load(constants.EnvData.ENV_PATH)
	if err != nil {
		log.Printf("error loading %s file", constants.EnvData.ENV_PATH)
	}

	if constants.EnvData.ADMIN_SECRET == "" {
		constants.EnvData.ADMIN_SECRET = os.Getenv("ADMIN_SECRET")
	}

	if constants.EnvData.DATABASE_TYPE == "" {
		constants.EnvData.DATABASE_TYPE = os.Getenv("DATABASE_TYPE")
		log.Println(constants.EnvData.DATABASE_TYPE)

		if ARG_DB_TYPE != nil && *ARG_DB_TYPE != "" {
			constants.EnvData.DATABASE_TYPE = *ARG_DB_TYPE
		}

		if constants.EnvData.DATABASE_TYPE == "" {
			panic("DATABASE_TYPE is required")
		}
	}

	if constants.EnvData.DATABASE_URL == "" {
		constants.EnvData.DATABASE_URL = os.Getenv("DATABASE_URL")

		if ARG_DB_URL != nil && *ARG_DB_URL != "" {
			constants.EnvData.DATABASE_URL = *ARG_DB_URL
		}

		if constants.EnvData.DATABASE_URL == "" {
			panic("DATABASE_URL is required")
		}
	}

	if constants.EnvData.DATABASE_NAME == "" {
		constants.EnvData.DATABASE_NAME = os.Getenv("DATABASE_NAME")
		if constants.EnvData.DATABASE_NAME == "" {
			constants.EnvData.DATABASE_NAME = "authorizer"
		}
	}

	if constants.EnvData.ENV == "" {
		constants.EnvData.ENV = os.Getenv("ENV")
		if constants.EnvData.ENV == "" {
			constants.EnvData.ENV = "production"
		}

		if constants.EnvData.ENV == "production" {
			constants.EnvData.IS_PROD = true
			os.Setenv("GIN_MODE", "release")
		} else {
			constants.EnvData.IS_PROD = false
		}
	}

	if constants.EnvData.SMTP_HOST == "" {
		constants.EnvData.SMTP_HOST = os.Getenv("SMTP_HOST")
	}

	if constants.EnvData.SMTP_PORT == "" {
		constants.EnvData.SMTP_PORT = os.Getenv("SMTP_PORT")
	}

	if constants.EnvData.SENDER_EMAIL == "" {
		constants.EnvData.SENDER_EMAIL = os.Getenv("SENDER_EMAIL")
	}

	if constants.EnvData.SENDER_PASSWORD == "" {
		constants.EnvData.SENDER_PASSWORD = os.Getenv("SENDER_PASSWORD")
	}

	if constants.EnvData.JWT_SECRET == "" {
		constants.EnvData.JWT_SECRET = os.Getenv("JWT_SECRET")
		if constants.EnvData.JWT_SECRET == "" {
			constants.EnvData.JWT_SECRET = uuid.New().String()
		}
	}

	if constants.EnvData.JWT_TYPE == "" {
		constants.EnvData.JWT_TYPE = os.Getenv("JWT_TYPE")
		if constants.EnvData.JWT_TYPE == "" {
			constants.EnvData.JWT_TYPE = "HS256"
		}
	}

	if constants.EnvData.JWT_ROLE_CLAIM == "" {
		constants.EnvData.JWT_ROLE_CLAIM = os.Getenv("JWT_ROLE_CLAIM")

		if constants.EnvData.JWT_ROLE_CLAIM == "" {
			constants.EnvData.JWT_ROLE_CLAIM = "role"
		}
	}

	if constants.EnvData.AUTHORIZER_URL == "" {
		constants.EnvData.AUTHORIZER_URL = strings.TrimSuffix(os.Getenv("AUTHORIZER_URL"), "/")

		if ARG_AUTHORIZER_URL != nil && *ARG_AUTHORIZER_URL != "" {
			constants.EnvData.AUTHORIZER_URL = *ARG_AUTHORIZER_URL
		}
	}

	if constants.EnvData.PORT == "" {
		constants.EnvData.PORT = os.Getenv("PORT")
		if constants.EnvData.PORT == "" {
			constants.EnvData.PORT = "8080"
		}
	}

	if constants.EnvData.REDIS_URL == "" {
		constants.EnvData.REDIS_URL = os.Getenv("REDIS_URL")
	}

	if constants.EnvData.COOKIE_NAME == "" {
		constants.EnvData.COOKIE_NAME = os.Getenv("COOKIE_NAME")
		if constants.EnvData.COOKIE_NAME == "" {
			constants.EnvData.COOKIE_NAME = "authorizer"
		}
	}

	if constants.EnvData.GOOGLE_CLIENT_ID == "" {
		constants.EnvData.GOOGLE_CLIENT_ID = os.Getenv("GOOGLE_CLIENT_ID")
	}

	if constants.EnvData.GOOGLE_CLIENT_SECRET == "" {
		constants.EnvData.GOOGLE_CLIENT_SECRET = os.Getenv("GOOGLE_CLIENT_SECRET")
	}

	if constants.EnvData.GITHUB_CLIENT_ID == "" {
		constants.EnvData.GITHUB_CLIENT_ID = os.Getenv("GITHUB_CLIENT_ID")
	}

	if constants.EnvData.GITHUB_CLIENT_SECRET == "" {
		constants.EnvData.GITHUB_CLIENT_SECRET = os.Getenv("GITHUB_CLIENT_SECRET")
	}

	if constants.EnvData.FACEBOOK_CLIENT_ID == "" {
		constants.EnvData.FACEBOOK_CLIENT_ID = os.Getenv("FACEBOOK_CLIENT_ID")
	}

	if constants.EnvData.FACEBOOK_CLIENT_SECRET == "" {
		constants.EnvData.FACEBOOK_CLIENT_SECRET = os.Getenv("FACEBOOK_CLIENT_SECRET")
	}

	if constants.EnvData.RESET_PASSWORD_URL == "" {
		constants.EnvData.RESET_PASSWORD_URL = strings.TrimPrefix(os.Getenv("RESET_PASSWORD_URL"), "/")
	}

	constants.EnvData.DISABLE_BASIC_AUTHENTICATION = os.Getenv("DISABLE_BASIC_AUTHENTICATION") == "true"
	constants.EnvData.DISABLE_EMAIL_VERIFICATION = os.Getenv("DISABLE_EMAIL_VERIFICATION") == "true"
	constants.EnvData.DISABLE_MAGIC_LINK_LOGIN = os.Getenv("DISABLE_MAGIC_LINK_LOGIN") == "true"
	constants.EnvData.DISABLE_LOGIN_PAGE = os.Getenv("DISABLE_LOGIN_PAGE") == "true"

	if constants.EnvData.SMTP_HOST == "" || constants.EnvData.SENDER_EMAIL == "" || constants.EnvData.SENDER_PASSWORD == "" {
		constants.EnvData.DISABLE_EMAIL_VERIFICATION = true
		constants.EnvData.DISABLE_MAGIC_LINK_LOGIN = true
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

	constants.EnvData.ALLOWED_ORIGINS = allowedOrigins

	if constants.EnvData.DISABLE_EMAIL_VERIFICATION {
		constants.EnvData.DISABLE_MAGIC_LINK_LOGIN = true
	}

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

	constants.EnvData.ROLES = roles
	constants.EnvData.DEFAULT_ROLES = defaultRoles
	constants.EnvData.PROTECTED_ROLES = protectedRoles

	if os.Getenv("ORGANIZATION_NAME") != "" {
		constants.EnvData.ORGANIZATION_NAME = os.Getenv("ORGANIZATION_NAME")
	}

	if os.Getenv("ORGANIZATION_LOGO") != "" {
		constants.EnvData.ORGANIZATION_LOGO = os.Getenv("ORGANIZATION_LOGO")
	}
}

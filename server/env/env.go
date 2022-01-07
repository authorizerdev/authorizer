package env

import (
	"log"
	"os"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/utils"
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
	if constants.ENV_PATH == "" {
		constants.ENV_PATH = `.env`
	}

	if ARG_ENV_FILE != nil && *ARG_ENV_FILE != "" {
		constants.ENV_PATH = *ARG_ENV_FILE
	}

	err := godotenv.Load(constants.ENV_PATH)
	if err != nil {
		log.Printf("error loading %s file", constants.ENV_PATH)
	}

	if constants.ADMIN_SECRET == "" {
		constants.ADMIN_SECRET = os.Getenv("ADMIN_SECRET")
		if constants.ADMIN_SECRET == "" {
			panic("root admin secret is required")
		}
	}

	if constants.ENV == "" {
		constants.ENV = os.Getenv("ENV")
		if constants.ENV == "" {
			constants.ENV = "production"
		}

		if constants.ENV == "production" {
			constants.IS_PROD = true
			os.Setenv("GIN_MODE", "release")
		} else {
			constants.IS_PROD = false
		}
	}

	if constants.DATABASE_TYPE == "" {
		constants.DATABASE_TYPE = os.Getenv("DATABASE_TYPE")
		log.Println(constants.DATABASE_TYPE)

		if ARG_DB_TYPE != nil && *ARG_DB_TYPE != "" {
			constants.DATABASE_TYPE = *ARG_DB_TYPE
		}

		if constants.DATABASE_TYPE == "" {
			panic("DATABASE_TYPE is required")
		}
	}

	if constants.DATABASE_URL == "" {
		constants.DATABASE_URL = os.Getenv("DATABASE_URL")

		if ARG_DB_URL != nil && *ARG_DB_URL != "" {
			constants.DATABASE_URL = *ARG_DB_URL
		}

		if constants.DATABASE_URL == "" {
			panic("DATABASE_URL is required")
		}
	}

	if constants.DATABASE_NAME == "" {
		constants.DATABASE_NAME = os.Getenv("DATABASE_NAME")
		if constants.DATABASE_NAME == "" {
			constants.DATABASE_NAME = "authorizer"
		}
	}

	if constants.SMTP_HOST == "" {
		constants.SMTP_HOST = os.Getenv("SMTP_HOST")
	}

	if constants.SMTP_PORT == "" {
		constants.SMTP_PORT = os.Getenv("SMTP_PORT")
	}

	if constants.SMTP_USERNAME == "" {
		constants.SMTP_USERNAME = os.Getenv("SMTP_USERNAME")
	}

	if constants.SMTP_PASSWORD == "" {
		constants.SMTP_PASSWORD = os.Getenv("SMTP_PASSWORD")
	}

	if constants.SENDER_EMAIL == "" {
		constants.SENDER_EMAIL = os.Getenv("SENDER_EMAIL")
	}

	if constants.JWT_SECRET == "" {
		constants.JWT_SECRET = os.Getenv("JWT_SECRET")
	}

	if constants.JWT_TYPE == "" {
		constants.JWT_TYPE = os.Getenv("JWT_TYPE")
	}

	if constants.JWT_ROLE_CLAIM == "" {
		constants.JWT_ROLE_CLAIM = os.Getenv("JWT_ROLE_CLAIM")

		if constants.JWT_ROLE_CLAIM == "" {
			constants.JWT_ROLE_CLAIM = "role"
		}
	}

	if constants.AUTHORIZER_URL == "" {
		constants.AUTHORIZER_URL = strings.TrimSuffix(os.Getenv("AUTHORIZER_URL"), "/")

		if ARG_AUTHORIZER_URL != nil && *ARG_AUTHORIZER_URL != "" {
			constants.AUTHORIZER_URL = *ARG_AUTHORIZER_URL
		}
	}

	if constants.PORT == "" {
		constants.PORT = os.Getenv("PORT")
		if constants.PORT == "" {
			constants.PORT = "8080"
		}
	}

	if constants.REDIS_URL == "" {
		constants.REDIS_URL = os.Getenv("REDIS_URL")
	}

	if constants.COOKIE_NAME == "" {
		constants.COOKIE_NAME = os.Getenv("COOKIE_NAME")
	}

	if constants.GOOGLE_CLIENT_ID == "" {
		constants.GOOGLE_CLIENT_ID = os.Getenv("GOOGLE_CLIENT_ID")
	}

	if constants.GOOGLE_CLIENT_SECRET == "" {
		constants.GOOGLE_CLIENT_SECRET = os.Getenv("GOOGLE_CLIENT_SECRET")
	}

	if constants.GITHUB_CLIENT_ID == "" {
		constants.GITHUB_CLIENT_ID = os.Getenv("GITHUB_CLIENT_ID")
	}

	if constants.GITHUB_CLIENT_SECRET == "" {
		constants.GITHUB_CLIENT_SECRET = os.Getenv("GITHUB_CLIENT_SECRET")
	}

	if constants.FACEBOOK_CLIENT_ID == "" {
		constants.FACEBOOK_CLIENT_ID = os.Getenv("FACEBOOK_CLIENT_ID")
	}

	if constants.FACEBOOK_CLIENT_SECRET == "" {
		constants.FACEBOOK_CLIENT_SECRET = os.Getenv("FACEBOOK_CLIENT_SECRET")
	}

	if constants.RESET_PASSWORD_URL == "" {
		constants.RESET_PASSWORD_URL = strings.TrimPrefix(os.Getenv("RESET_PASSWORD_URL"), "/")
	}

	constants.DISABLE_BASIC_AUTHENTICATION = os.Getenv("DISABLE_BASIC_AUTHENTICATION") == "true"
	constants.DISABLE_EMAIL_VERIFICATION = os.Getenv("DISABLE_EMAIL_VERIFICATION") == "true"
	constants.DISABLE_MAGIC_LINK_LOGIN = os.Getenv("DISABLE_MAGIC_LINK_LOGIN") == "true"
	constants.DISABLE_LOGIN_PAGE = os.Getenv("DISABLE_LOGIN_PAGE") == "true"

	if constants.SMTP_HOST == "" || constants.SMTP_USERNAME == "" || constants.SMTP_PASSWORD == "" || constants.SENDER_EMAIL == "" {
		constants.DISABLE_EMAIL_VERIFICATION = true
		constants.DISABLE_MAGIC_LINK_LOGIN = true
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

	constants.ALLOWED_ORIGINS = allowedOrigins

	if constants.JWT_TYPE == "" {
		constants.JWT_TYPE = "HS256"
	}

	if constants.COOKIE_NAME == "" {
		constants.COOKIE_NAME = "authorizer"
	}

	if constants.DISABLE_EMAIL_VERIFICATION {
		constants.DISABLE_MAGIC_LINK_LOGIN = true
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

	if len(roles) > 0 && len(defaultRoles) == 0 && len(defaultRoleSplit) > 0 {
		panic(`Invalid DEFAULT_ROLE environment variable. It can be one from give ROLES environment variable value`)
	}

	constants.ROLES = roles
	constants.DEFAULT_ROLES = defaultRoles
	constants.PROTECTED_ROLES = protectedRoles

	if os.Getenv("ORGANIZATION_NAME") != "" {
		constants.ORGANIZATION_NAME = os.Getenv("ORGANIZATION_NAME")
	}

	if os.Getenv("ORGANIZATION_LOGO") != "" {
		constants.ORGANIZATION_LOGO = os.Getenv("ORGANIZATION_LOGO")
	}
}

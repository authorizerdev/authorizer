package env

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/joho/godotenv"
)

// build variables
var (
	VERSION            string
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
	ARG_DB_URL = flag.String("database_url", "", "Database connection string")
	ARG_DB_TYPE = flag.String("database_type", "", "Database type, possible values are postgres,mysql,sqlite")
	ARG_AUTHORIZER_URL = flag.String("authorizer_url", "", "URL for authorizer instance, eg: https://xyz.herokuapp.com")
	ARG_ENV_FILE = flag.String("env_file", "", "Env file path")

	flag.Parse()
	if *ARG_ENV_FILE != "" {
		constants.ENV_PATH = *ARG_ENV_FILE
	}

	err := godotenv.Load(constants.ENV_PATH)
	if err != nil {
		log.Printf("error loading %s file", constants.ENV_PATH)
	}

	constants.VERSION = VERSION
	constants.ADMIN_SECRET = os.Getenv("ADMIN_SECRET")
	constants.ENV = os.Getenv("ENV")
	constants.DATABASE_TYPE = os.Getenv("DATABASE_TYPE")
	constants.DATABASE_URL = os.Getenv("DATABASE_URL")
	constants.DATABASE_NAME = os.Getenv("DATABASE_NAME")
	constants.SMTP_HOST = os.Getenv("SMTP_HOST")
	constants.SMTP_PORT = os.Getenv("SMTP_PORT")
	constants.SENDER_EMAIL = os.Getenv("SENDER_EMAIL")
	constants.SENDER_PASSWORD = os.Getenv("SENDER_PASSWORD")
	constants.JWT_SECRET = os.Getenv("JWT_SECRET")
	constants.JWT_TYPE = os.Getenv("JWT_TYPE")
	constants.AUTHORIZER_URL = strings.TrimSuffix(os.Getenv("AUTHORIZER_URL"), "/")
	constants.PORT = os.Getenv("PORT")
	constants.REDIS_URL = os.Getenv("REDIS_URL")
	constants.COOKIE_NAME = os.Getenv("COOKIE_NAME")
	constants.GOOGLE_CLIENT_ID = os.Getenv("GOOGLE_CLIENT_ID")
	constants.GOOGLE_CLIENT_SECRET = os.Getenv("GOOGLE_CLIENT_SECRET")
	constants.GITHUB_CLIENT_ID = os.Getenv("GITHUB_CLIENT_ID")
	constants.GITHUB_CLIENT_SECRET = os.Getenv("GITHUB_CLIENT_SECRET")
	constants.FACEBOOK_CLIENT_ID = os.Getenv("FACEBOOK_CLIENT_ID")
	constants.FACEBOOK_CLIENT_SECRET = os.Getenv("FACEBOOK_CLIENT_SECRET")
	constants.TWITTER_CLIENT_ID = os.Getenv("TWITTER_CLIENT_ID")
	constants.TWITTER_CLIENT_SECRET = os.Getenv("TWITTER_CLIENT_SECRET")
	constants.RESET_PASSWORD_URL = strings.TrimPrefix(os.Getenv("RESET_PASSWORD_URL"), "/")
	constants.DISABLE_BASIC_AUTHENTICATION = os.Getenv("DISABLE_BASIC_AUTHENTICATION") == "true"
	constants.DISABLE_EMAIL_VERIFICATION = os.Getenv("DISABLE_EMAIL_VERIFICATION") == "true"
	constants.DISABLE_MAGIC_LOGIN = os.Getenv("DISABLE_MAGIC_LOGIN") == "true"
	constants.JWT_ROLE_CLAIM = os.Getenv("JWT_ROLE_CLAIM")

	if constants.ADMIN_SECRET == "" {
		panic("root admin secret is required")
	}

	if constants.ENV == "" {
		constants.ENV = "production"
	}

	if constants.ENV == "production" {
		constants.IS_PROD = true
		os.Setenv("GIN_MODE", "release")
	} else {
		constants.IS_PROD = false
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

	if *ARG_AUTHORIZER_URL != "" {
		constants.AUTHORIZER_URL = *ARG_AUTHORIZER_URL
	}

	if *ARG_DB_URL != "" {
		constants.DATABASE_URL = *ARG_DB_URL
	}

	if *ARG_DB_TYPE != "" {
		constants.DATABASE_TYPE = *ARG_DB_TYPE
	}

	if constants.DATABASE_URL == "" {
		panic("Database url is required")
	}

	if constants.DATABASE_TYPE == "" {
		panic("Database type is required")
	}

	if constants.DATABASE_NAME == "" {
		constants.DATABASE_NAME = "authorizer"
	}

	if constants.JWT_TYPE == "" {
		constants.JWT_TYPE = "HS256"
	}

	if constants.COOKIE_NAME == "" {
		constants.COOKIE_NAME = "authorizer"
	}

	if constants.SMTP_HOST == "" || constants.SENDER_EMAIL == "" || constants.SENDER_PASSWORD == "" {
		constants.DISABLE_EMAIL_VERIFICATION = true
		constants.DISABLE_MAGIC_LOGIN = true
	}

	if constants.DISABLE_EMAIL_VERIFICATION {
		constants.DISABLE_MAGIC_LOGIN = true
	}

	rolesSplit := strings.Split(os.Getenv("ROLES"), ",")
	roles := []string{}
	if len(rolesSplit) == 0 {
		roles = []string{"user"}
	}

	defaultRoleSplit := strings.Split(os.Getenv("DEFAULT_ROLES"), ",")
	defaultRoles := []string{}

	if len(defaultRoleSplit) == 0 {
		defaultRoles = []string{"user"}
	}

	protectedRolesSplit := strings.Split(os.Getenv("PROTECTED_ROLES"), ",")
	protectedRoles := []string{}

	if len(protectedRolesSplit) > 0 {
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

	if constants.JWT_ROLE_CLAIM == "" {
		constants.JWT_ROLE_CLAIM = "role"
	}

	if os.Getenv("ORGANIZATION_NAME") != "" {
		constants.ORGANIZATION_NAME = os.Getenv("ORGANIZATION_NAME")
	}

	if os.Getenv("ORGANIZATION_LOGO") != "" {
		constants.ORGANIZATION_LOGO = os.Getenv("ORGANIZATION_LOGO")
	}
}

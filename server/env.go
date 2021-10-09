package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/joho/godotenv"
)

// build variables
var (
	Version            string
	ARG_DB_URL         *string
	ARG_DB_TYPE        *string
	ARG_AUTHORIZER_URL *string
	ARG_ENV_FILE       *string
)

// InitEnv -> to initialize env and through error if required env are not present
func InitEnv() {
	envPath := `.env`
	ARG_DB_URL = flag.String("database_url", "", "Database connection string")
	ARG_DB_TYPE = flag.String("database_type", "", "Database type, possible values are postgres,mysql,sqlite")
	ARG_AUTHORIZER_URL = flag.String("authorizer_url", "", "URL for authorizer instance, eg: https://xyz.herokuapp.com")
	ARG_ENV_FILE = flag.String("env_file", "", "Env file path")

	flag.Parse()
	if *ARG_ENV_FILE != "" {
		envPath = *ARG_ENV_FILE
	}

	err := godotenv.Load(envPath)
	if err != nil {
		log.Println("Error loading .env file")
	}

	constants.VERSION = Version
	constants.ADMIN_SECRET = os.Getenv("ADMIN_SECRET")
	constants.ENV = os.Getenv("ENV")
	constants.DATABASE_TYPE = os.Getenv("DATABASE_TYPE")
	constants.DATABASE_URL = os.Getenv("DATABASE_URL")
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
	constants.DISABLE_BASIC_AUTHENTICATION = os.Getenv("DISABLE_BASIC_AUTHENTICATION")
	constants.DISABLE_EMAIL_VERIFICATION = os.Getenv("DISABLE_EMAIL_VERIFICATION")
	constants.DEFAULT_ROLE = os.Getenv("DEFAULT_ROLE")
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
	for _, val := range allowedOriginsSplit {
		trimVal := strings.TrimSpace(val)
		if trimVal != "" {
			allowedOrigins = append(allowedOrigins, trimVal)
		}
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

	if constants.JWT_TYPE == "" {
		constants.JWT_TYPE = "HS256"
	}

	if constants.COOKIE_NAME == "" {
		constants.COOKIE_NAME = "authorizer"
	}

	if constants.DISABLE_BASIC_AUTHENTICATION == "" {
		constants.DISABLE_BASIC_AUTHENTICATION = "false"
	}

	if constants.DISABLE_EMAIL_VERIFICATION == "" && constants.DISABLE_BASIC_AUTHENTICATION == "false" {
		if constants.SMTP_HOST == "" || constants.SENDER_EMAIL == "" || constants.SENDER_PASSWORD == "" {
			constants.DISABLE_EMAIL_VERIFICATION = "true"
		} else {
			constants.DISABLE_EMAIL_VERIFICATION = "false"
		}
	}

	rolesSplit := strings.Split(os.Getenv("ROLES"), ",")
	roles := []string{}
	defaultRole := ""

	for _, val := range rolesSplit {
		trimVal := strings.TrimSpace(val)
		if trimVal != "" {
			roles = append(roles, trimVal)
		}

		if trimVal == constants.DEFAULT_ROLE {
			defaultRole = trimVal
		}
	}
	if len(roles) > 0 && defaultRole == "" {
		panic(`Invalid DEFAULT_ROLE environment variable. It can be one from give ROLES environment variable value`)
	}

	if len(roles) == 0 {
		roles = []string{"user", "admin"}
		constants.DEFAULT_ROLE = "user"
	}

	constants.ROLES = roles

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

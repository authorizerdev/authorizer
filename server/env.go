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
var Version string

// ParseArgs -> to parse the cli flag and get db url. This is useful with heroku button
func ParseArgs() {
	dbURL := flag.String("database_url", "", "Database connection string")
	dbType := flag.String("databse_type", "", "Database type, possible values are postgres,mysql,sqlite")
	authorizerURL := flag.String("authorizer_url", "", "URL for authorizer instance, eg: https://xyz.herokuapp.com")

	flag.Parse()
	if *dbURL != "" {
		constants.DATABASE_URL = *dbURL
	}

	if *dbType != "" {
		constants.DATABASE_TYPE = *dbType
	}

	if *authorizerURL != "" {
		constants.AUTHORIZER_URL = *authorizerURL
	}
}

// InitEnv -> to initialize env and through error if required env are not present
func InitEnv() {
	envPath := `.env`
	envFile := flag.String("env_file", "", "Env file path")
	flag.Parse()
	if *envFile != "" {
		envPath = *envFile
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

	allowedCallbackSplit := strings.Split(os.Getenv("ALLOWED_CALLBACK_URLS"), ",")
	allowedCallbacks := []string{}
	for _, val := range allowedCallbackSplit {
		trimVal := strings.TrimSpace(val)
		if trimVal != "" {
			allowedCallbacks = append(allowedCallbacks, trimVal)
		}
	}
	if len(allowedCallbackSplit) == 0 {
		allowedCallbackSplit = []string{"*"}
	}
	constants.ALLOWED_CALLBACK_URLS = allowedCallbackSplit

	ParseArgs()
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
		panic(`Invalid DEFAULT_ROLE environment. It can be one from give ROLES environment variable value`)
	}

	if len(roles) == 0 {
		roles = []string{"user", "admin"}
		constants.DEFAULT_ROLE = "user"
	}

	constants.ROLES = roles
}

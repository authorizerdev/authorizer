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
	dbType := flag.String("databse_type", "", "Database type, possible values are postgres,mysql,sqlit")
	flag.Parse()
	if *dbURL != "" {
		constants.DATABASE_URL = *dbURL
	}

	if *dbType != "" {
		constants.DATABASE_TYPE = *dbType
	}
}

// InitEnv -> to initialize env and through error if required env are not present
func InitEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	constants.VERSION = Version

	constants.ROOT_SECRET = os.Getenv("ROOT_SECRET")
	constants.ENV = os.Getenv("ENV")
	constants.DATABASE_TYPE = os.Getenv("DATABASE_TYPE")
	constants.DATABASE_URL = os.Getenv("DATABASE_URL")
	constants.SMTP_HOST = os.Getenv("SMTP_HOST")
	constants.SMTP_PORT = os.Getenv("SMTP_PORT")
	constants.SENDER_EMAIL = os.Getenv("SENDER_EMAIL")
	constants.SENDER_PASSWORD = os.Getenv("SENDER_PASSWORD")
	constants.JWT_SECRET = os.Getenv("JWT_SECRET")
	constants.JWT_TYPE = os.Getenv("JWT_TYPE")
	constants.FRONTEND_URL = strings.TrimSuffix(os.Getenv("FRONTEND_URL"), "/")
	constants.AUTHORIZER_DOMAIN = strings.TrimSuffix(os.Getenv("AUTHORIZER_DOMAIN"), "/")
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
	constants.FORGOT_PASSWORD_URI = strings.TrimPrefix(os.Getenv("FORGOT_PASSWORD_URI"), "/")
	constants.VERIFY_EMAIL_URI = strings.TrimPrefix(os.Getenv("VERIFY_EMAIL_URI"), "/")
	constants.DISABLE_BASIC_AUTHENTICATION = os.Getenv("DISABLE_BASIC_AUTHENTICATION")
	constants.DISABLE_EMAIL_VERICATION = os.Getenv("DISABLE_EMAIL_VERICATION")

	if constants.ROOT_SECRET == "" {
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

	if constants.AUTHORIZER_DOMAIN == "" {
		constants.AUTHORIZER_DOMAIN = "http://localhost:8080"
	}

	if constants.DISABLE_BASIC_AUTHENTICATION == "" {
		constants.DISABLE_BASIC_AUTHENTICATION = "false"
	}

	if constants.DISABLE_EMAIL_VERICATION == "" && constants.DISABLE_BASIC_AUTHENTICATION == "false" {
		if constants.SMTP_HOST == "" || constants.SENDER_EMAIL == "" || constants.SENDER_PASSWORD == "" {
			constants.DISABLE_EMAIL_VERICATION = "true"
		} else {
			constants.DISABLE_EMAIL_VERICATION = "false"
		}
	}
}

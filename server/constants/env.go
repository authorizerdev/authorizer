package constants

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/yauthdev/yauth/server/enum"
)

var (
	YAUTH_ADMIN_SECRET   = ""
	ENV                  = ""
	DB_TYPE              = ""
	DB_URL               = ""
	SMTP_HOST            = ""
	SMTP_PORT            = ""
	SENDER_EMAIL         = ""
	SENDER_PASSWORD      = ""
	JWT_TYPE             = ""
	JWT_SECRET           = ""
	FRONTEND_URL         = ""
	SERVER_URL           = ""
	PORT                 = "8080"
	REDIS_URL            = ""
	IS_PROD              = false
	COOKIE_NAME          = ""
	GOOGLE_CLIENT_ID     = ""
	GOOGLE_CLIENT_SECRET = ""
	GITHUB_CLIENT_ID     = ""
	GITHUB_CLIENT_SECRET = ""
	// FACEBOOK_CLIENT_ID     = ""
	// FACEBOOK_CLIENT_SECRET = ""
	FORGOT_PASSWORD_URI = ""
	VERIFY_EMAIL_URI    = ""
)

func ParseArgs() {
	dbURL := flag.String("db_url", "", "Database connection string")
	dbType := flag.String("db_type", "", "Database type, possible values are postgres,mysql,sqlit")
	flag.Parse()
	if *dbURL != "" {
		DB_URL = *dbURL
	}

	if *dbType != "" {
		DB_TYPE = *dbType
	}
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	YAUTH_ADMIN_SECRET = os.Getenv("YAUTH_ADMIN_SECRET")
	ENV = os.Getenv("ENV")
	DB_TYPE = os.Getenv("DB_TYPE")
	DB_URL = os.Getenv("DB_URL")
	SMTP_HOST = os.Getenv("SMTP_HOST")
	SMTP_PORT = os.Getenv("SMTP_PORT")
	SENDER_EMAIL = os.Getenv("SENDER_EMAIL")
	SENDER_PASSWORD = os.Getenv("SENDER_PASSWORD")
	JWT_SECRET = os.Getenv("JWT_SECRET")
	JWT_TYPE = os.Getenv("JWT_TYPE")
	FRONTEND_URL = strings.TrimSuffix(os.Getenv("FRONTEND_URL"), "/")
	SERVER_URL = strings.TrimSuffix(os.Getenv("SERVER_URL"), "/")
	PORT = os.Getenv("PORT")
	REDIS_URL = os.Getenv("REDIS_URL")
	COOKIE_NAME = os.Getenv("COOKIE_NAME")
	GOOGLE_CLIENT_ID = os.Getenv("GOOGLE_CLIENT_ID")
	GOOGLE_CLIENT_SECRET = os.Getenv("GOOGLE_CLIENT_SECRET")
	GITHUB_CLIENT_ID = os.Getenv("GITHUB_CLIENT_ID")
	GITHUB_CLIENT_SECRET = os.Getenv("GITHUB_CLIENT_SECRET")
	// FACEBOOK_CLIENT_ID = os.Getenv("FACEBOOK_CLIENT_ID")
	// FACEBOOK_CLIENT_SECRET = os.Getenv("FACEBOOK_CLIENT_SECRET")
	FORGOT_PASSWORD_URI = strings.TrimPrefix(os.Getenv("FORGOT_PASSWORD_URI"), "/")
	VERIFY_EMAIL_URI = strings.TrimPrefix(os.Getenv("VERIFY_EMAIL_URI"), "/")
	if YAUTH_ADMIN_SECRET == "" {
		panic("Yauth admin secret is required")
	}

	if ENV == "" {
		ENV = "production"
	}

	if ENV == "production" {
		IS_PROD = true
	} else {
		IS_PROD = false
	}

	ParseArgs()

	if DB_TYPE == "" {
		DB_TYPE = enum.Postgres.String()
	}

	if DB_URL == "" {
		DB_TYPE = "postgresql://localhost:5432/postgres"
	}

	if JWT_TYPE == "" {
		JWT_TYPE = "HS256"
	}

	if COOKIE_NAME == "" {
		COOKIE_NAME = "yauth"
	}

	if SERVER_URL == "" {
		SERVER_URL = "http://localhost:8080"
	}
}

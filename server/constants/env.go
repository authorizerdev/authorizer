package constants

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/yauthdev/yauth/server/enum"
)

var (
	ENV                    = ""
	DB_TYPE                = ""
	DB_URL                 = ""
	SMTP_HOST              = ""
	SMTP_PORT              = ""
	SENDER_EMAIL           = ""
	SENDER_PASSWORD        = ""
	JWT_TYPE               = ""
	JWT_SECRET             = ""
	FRONTEND_URL           = ""
	SERVER_URL             = ""
	PORT                   = "8080"
	REDIS_URL              = ""
	IS_PROD                = false
	COOKIE_NAME            = ""
	GOOGLE_CLIENT_ID       = ""
	GOOGLE_CLIENT_SECRET   = ""
	GITHUB_CLIENT_ID       = ""
	GITHUB_CLIENT_SECRET   = ""
	FACEBOOK_CLIENT_ID     = ""
	FACEBOOK_CLIENT_SECRET = ""
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

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
	FACEBOOK_CLIENT_ID = os.Getenv("FACEBOOK_CLIENT_ID")
	FACEBOOK_CLIENT_SECRET = os.Getenv("FACEBOOK_CLIENT_SECRET")

	if ENV == "" {
		ENV = "production"
	}

	if ENV == "production" {
		IS_PROD = true
	} else {
		IS_PROD = false
	}

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

package constants

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/yauthdev/yauth/server/enum"
)

var (
	ENV             = ""
	DB_TYPE         = ""
	DB_URL          = ""
	SMTP_HOST       = ""
	SMTP_PORT       = ""
	SENDER_EMAIL    = ""
	SENDER_PASSWORD = ""
	JWT_TYPE        = ""
	JWT_SECRET      = ""
	FRONTEND_URL    = ""
	PORT            = "8080"
	REDIS_URL       = ""
	IS_PROD         = false
	COOKIE_NAME     = ""
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
	FRONTEND_URL = os.Getenv("FRONTEND_URL")
	PORT = os.Getenv("PORT")
	REDIS_URL = os.Getenv("REDIS_URL")
	COOKIE_NAME = os.Getenv("COOKIE_NAME")

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
}

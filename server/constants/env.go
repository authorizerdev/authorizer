package constants

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/yauthdev/yauth/server/enum"
)

var (
	DB_TYPE         = enum.Postgres
	DB_URL          = "postgresql://localhost:5432/postgres"
	SMTP_HOST       = ""
	SMTP_PORT       = ""
	SENDER_EMAIL    = ""
	SENDER_PASSWORD = ""
	JWT_TYPE        = ""
	JWT_SECRET      = ""
	FRONTEND_URL    = ""
	PORT            = "8080"
	REDIS_URL       = ""
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	SMTP_HOST = os.Getenv("SMTP_HOST")
	SMTP_PORT = os.Getenv("SMTP_PORT")
	SENDER_EMAIL = os.Getenv("SENDER_EMAIL")
	SENDER_PASSWORD = os.Getenv("SENDER_PASSWORD")
	JWT_SECRET = os.Getenv("JWT_SECRET")
	JWT_TYPE = os.Getenv("JWT_TYPE")
	FRONTEND_URL = os.Getenv("FRONTEND_URL")
	PORT = os.Getenv("PORT")
	REDIS_URL = os.Getenv("REDIS_URL")

	if JWT_TYPE == "" {
		JWT_TYPE = "HS256"
	}
}

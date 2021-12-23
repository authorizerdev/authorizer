package main

import (
	"flag"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/handlers"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/router"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
)

func main() {
	env.ARG_DB_URL = flag.String("database_url", "", "Database connection string")
	env.ARG_DB_TYPE = flag.String("database_type", "", "Database type, possible values are postgres,mysql,sqlite")
	env.ARG_AUTHORIZER_URL = flag.String("authorizer_url", "", "URL for authorizer instance, eg: https://xyz.herokuapp.com")
	env.ARG_ENV_FILE = flag.String("env_file", "", "Env file path")
	flag.Parse()

	env.InitEnv()
	db.InitDB()
	session.InitSession()
	oauth.InitOAuth()
	utils.InitServer()

	router := router.InitRouter()

	// login wall app related routes.
	// if we put them in router file then tests would fail as templates or build path will be different
	router.LoadHTMLGlob("templates/*")
	app := router.Group("/app")
	{
		app.Static("/build", "app/build")
		app.GET("/", handlers.AppHandler())
		app.GET("/reset-password", handlers.AppHandler())
	}
	router.Run(":" + constants.PORT)
}

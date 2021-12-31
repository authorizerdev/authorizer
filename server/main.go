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

var VERSION string

func main() {
	env.ARG_DB_URL = flag.String("database_url", "", "Database connection string")
	env.ARG_DB_TYPE = flag.String("database_type", "", "Database type, possible values are postgres,mysql,sqlite")
	env.ARG_AUTHORIZER_URL = flag.String("authorizer_url", "", "URL for authorizer instance, eg: https://xyz.herokuapp.com")
	env.ARG_ENV_FILE = flag.String("env_file", "", "Env file path")
	flag.Parse()

	constants.EnvData.VERSION = VERSION

	env.InitEnv()
	db.InitDB()
	env.PersistEnv()

	session.InitSession()
	oauth.InitOAuth()
	utils.InitServer()

	router := router.InitRouter()

	router.LoadHTMLGlob("templates/*")
	// login page app related routes.
	// if we put them in router file then tests would fail as templates or build path will be different
	if !constants.EnvData.DISABLE_LOGIN_PAGE {
		app := router.Group("/app")
		{
			app.Static("/build", "app/build")
			app.GET("/", handlers.AppHandler())
			app.GET("/reset-password", handlers.AppHandler())
		}
	}

	app := router.Group("/dashboard")
	{
		app.Static("/build", "dashboard/build")
		app.GET("/", handlers.DashboardHandler())
	}

	router.Run(":" + constants.EnvData.PORT)
}

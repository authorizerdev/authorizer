package main

import (
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
	env.InitEnv()
	db.InitDB()
	session.InitSession()
	oauth.InitOAuth()
	utils.InitServer()

	router := router.InitRouter()

	// login wall app related routes
	router.LoadHTMLGlob("templates/*")
	app := router.Group("/app")
	{
		app.Static("/build", "app/build")
		app.GET("/", handlers.AppHandler())
		app.GET("/reset-password", handlers.AppHandler())
	}
	router.Run(":" + constants.PORT)
}

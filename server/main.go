package main

import (
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/env"
	"github.com/authorizerdev/authorizer/server/handlers"
	"github.com/authorizerdev/authorizer/server/middlewares"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

func main() {
	env.InitEnv()
	db.InitDB()
	session.InitSession()
	oauth.InitOAuth()
	utils.InitServer()

	r := gin.Default()
	r.Use(location.Default())
	r.Use(middlewares.GinContextToContextMiddleware())
	r.Use(middlewares.CORSMiddleware())

	r.GET("/", handlers.PlaygroundHandler())
	r.POST("/graphql", handlers.GraphqlHandler())
	r.GET("/verify_email", handlers.VerifyEmailHandler())
	r.GET("/oauth_login/:oauth_provider", handlers.OAuthLoginHandler())
	r.GET("/oauth_callback/:oauth_provider", handlers.OAuthCallbackHandler())

	// login wall app related routes

	r.LoadHTMLGlob("templates/*")
	app := r.Group("/app")
	{
		app.Static("/build", "app/build")
		app.GET("/", handlers.AppHandler())
		app.GET("/reset-password", handlers.AppHandler())
	}

	r.Run(":" + constants.PORT)
}

package router

import (
	"github.com/authorizerdev/authorizer/server/handlers"
	"github.com/authorizerdev/authorizer/server/middlewares"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	router := gin.Default()
	router.Use(location.Default())
	router.Use(middlewares.GinContextToContextMiddleware())
	router.Use(middlewares.CORSMiddleware())

	router.GET("/", handlers.PlaygroundHandler())
	router.POST("/graphql", handlers.GraphqlHandler())
	router.GET("/verify_email", handlers.VerifyEmailHandler())
	router.GET("/oauth_login/:oauth_provider", handlers.OAuthLoginHandler())
	router.GET("/oauth_callback/:oauth_provider", handlers.OAuthCallbackHandler())

	return router
}

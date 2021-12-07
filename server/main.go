package main

import (
	"context"
	"log"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/handlers"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
)

func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if constants.AUTHORIZER_URL == "" {
			url := location.Get(c)
			constants.AUTHORIZER_URL = url.Scheme + "://" + c.Request.Host
			log.Println("=> setting url:", constants.AUTHORIZER_URL)
		}
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// TODO use allowed origins for cors origin
// TODO throw error if url is not allowed
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		constants.APP_URL = origin

		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func main() {
	InitEnv()
	db.InitDB()
	session.InitSession()
	oauth.InitOAuth()
	utils.InitServer()

	r := gin.Default()
	r.Use(location.Default())
	r.Use(GinContextToContextMiddleware())
	r.Use(CORSMiddleware())

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

	r.Run()
}

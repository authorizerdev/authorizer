package main

import (
	"context"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/handlers"
	"github.com/authorizerdev/authorizer/server/oauth"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/gin-gonic/gin"
)

func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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

	r := gin.Default()
	r.Use(GinContextToContextMiddleware())
	r.Use(CORSMiddleware())

	r.GET("/", handlers.PlaygroundHandler())
	r.POST("/graphql", handlers.GraphqlHandler())
	r.GET("/verify_email", handlers.VerifyEmailHandler())
	r.GET("/oauth_login/:oauth_provider", handlers.OAuthLoginHandler())
	r.GET("/oauth_callback/:oauth_provider", handlers.OAuthCallbackHandler())

	// login wall app related routes
	r.Static("/app/build", "app/build")
	r.LoadHTMLGlob("templates/*")
	r.GET("/app", handlers.AppHandler())

	r.Run()
}

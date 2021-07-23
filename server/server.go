package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/yauthdev/yauth/server/enum"
	"github.com/yauthdev/yauth/server/handlers"
	"github.com/yauthdev/yauth/server/oauth"
)

func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		log.Println("-> origin", origin)
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
	r := gin.Default()
	r.Use(GinContextToContextMiddleware())
	r.Use(CORSMiddleware())
	r.GET("/", handlers.PlaygroundHandler())
	r.POST("/graphql", handlers.GraphqlHandler())
	r.GET("/verify_email", handlers.VerifyEmailHandler())
	if oauth.OAuthProvider.GoogleConfig != nil {
		r.GET("/login/google", handlers.OAuthLoginHandler(enum.GoogleProvider))
		r.GET("/callback/google", handlers.OAuthCallbackHandler(enum.GoogleProvider))
	}
	if oauth.OAuthProvider.GithubConfig != nil {
		r.GET("/login/github", handlers.OAuthLoginHandler(enum.GithubProvider))
		r.GET("/callback/github", handlers.OAuthCallbackHandler(enum.GithubProvider))
	}
	r.Run()
}

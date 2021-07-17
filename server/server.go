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

func main() {
	r := gin.Default()
	r.Use(GinContextToContextMiddleware())
	r.GET("/", handlers.PlaygroundHandler())
	r.POST("/graphql", handlers.GraphqlHandler())
	if oauth.OAuthProvider.GoogleConfig != nil {
		r.GET("/login/google", handlers.HandleOAuthLogin(enum.GoogleProvider))
		r.GET("/callback/google", handlers.HandleOAuthCallback(enum.GoogleProvider))
	}
	log.Println(oauth.OAuthProvider.GithubConfig)
	if oauth.OAuthProvider.GithubConfig != nil {
		r.GET("/login/github", handlers.HandleOAuthLogin(enum.GithubProvider))
		r.GET("/callback/github", handlers.HandleOAuthCallback(enum.GithubProvider))
	}
	r.Run()
}

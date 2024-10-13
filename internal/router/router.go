package router

import (
	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/handlers"
	"github.com/authorizerdev/authorizer/internal/middlewares"
)

// NewRouter creates new gin router
func NewRouter() *gin.Engine {
	router := gin.New()

	// router.Use(middlewares.Logger(log), gin.Recovery())
	router.Use(middlewares.GinContextToContextMiddleware())
	router.Use(middlewares.CORSMiddleware())
	router.Use(middlewares.ClientCheckMiddleware())

	router.GET("/", handlers.RootHandler())
	router.GET("/health", handlers.HealthHandler())
	router.POST("/graphql", handlers.GraphqlHandler())
	router.GET("/playground", handlers.PlaygroundHandler())
	router.GET("/oauth_login/:oauth_provider", handlers.OAuthLoginHandler())
	router.GET("/oauth_callback/:oauth_provider", handlers.OAuthCallbackHandler())
	router.POST("/oauth_callback/:oauth_provider", handlers.OAuthCallbackHandler())
	router.GET("/verify_email", handlers.VerifyEmailHandler())
	// OPEN ID routes
	router.GET("/.well-known/openid-configuration", handlers.OpenIDConfigurationHandler())
	router.GET("/.well-known/jwks.json", handlers.JWKsHandler())
	router.GET("/authorize", handlers.AuthorizeHandler())
	router.GET("/userinfo", handlers.UserInfoHandler())
	router.GET("/logout", handlers.LogoutHandler())
	router.POST("/oauth/token", handlers.TokenHandler())
	router.POST("/oauth/revoke", handlers.RevokeRefreshTokenHandler())

	router.LoadHTMLGlob("templates/*")
	// login page app related routes.
	app := router.Group("/app")
	{
		app.Static("/favicon_io", "app/favicon_io")
		app.Static("/build", "app/build")
		app.GET("/", handlers.AppHandler())
		app.GET("/:page", handlers.AppHandler())
	}

	// dashboard related routes
	dashboard := router.Group("/dashboard")
	{
		dashboard.Static("/favicon_io", "dashboard/favicon_io")
		dashboard.Static("/build", "dashboard/build")
		dashboard.Static("/public", "dashboard/public")
		dashboard.GET("/", handlers.DashboardHandler())
		dashboard.GET("/:page", handlers.DashboardHandler())
	}
	return router
}

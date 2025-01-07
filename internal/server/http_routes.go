package server

import (
	"github.com/gin-gonic/gin"
)

// NewRouter creates new gin router
func (s *server) NewRouter() *gin.Engine {
	router := gin.New()

	router.Use(s.Dependencies.HTTPProvider.LoggerMiddleware())
	router.Use(s.Dependencies.HTTPProvider.ContextMiddleware())
	router.Use(s.Dependencies.HTTPProvider.CORSMiddleware())
	router.Use(s.Dependencies.HTTPProvider.ClientCheckMiddleware())

	router.GET("/", s.Dependencies.HTTPProvider.RootHandler())
	router.GET("/health", s.Dependencies.HTTPProvider.HealthHandler())
	router.POST("/graphql", s.Dependencies.HTTPProvider.GraphqlHandler())
	router.GET("/playground", s.Dependencies.HTTPProvider.PlaygroundHandler())
	router.GET("/oauth_login/:oauth_provider", s.Dependencies.HTTPProvider.OAuthLoginHandler())
	router.GET("/oauth_callback/:oauth_provider", s.Dependencies.HTTPProvider.OAuthCallbackHandler())
	router.POST("/oauth_callback/:oauth_provider", s.Dependencies.HTTPProvider.OAuthCallbackHandler())
	router.GET("/verify_email", s.Dependencies.HTTPProvider.VerifyEmailHandler())
	// OPEN ID routes
	router.GET("/.well-known/openid-configuration", s.Dependencies.HTTPProvider.OpenIDConfigurationHandler())
	router.GET("/.well-known/jwks.json", s.Dependencies.HTTPProvider.JWKsHandler())
	router.GET("/authorize", s.Dependencies.HTTPProvider.AuthorizeHandler())
	router.GET("/userinfo", s.Dependencies.HTTPProvider.UserInfoHandler())
	router.GET("/logout", s.Dependencies.HTTPProvider.LogoutHandler())
	router.POST("/oauth/token", s.Dependencies.HTTPProvider.TokenHandler())
	router.POST("/oauth/revoke", s.Dependencies.HTTPProvider.RevokeRefreshTokenHandler())

	router.LoadHTMLGlob("web/templates/*")
	// login page app related routes.
	app := router.Group("/app")
	{
		app.Static("/favicon_io", "app/favicon_io")
		app.Static("/build", "app/build")
		app.GET("/", s.Dependencies.HTTPProvider.AppHandler())
		app.GET("/:page", s.Dependencies.HTTPProvider.AppHandler())
	}

	// dashboard related routes
	dashboard := router.Group("/dashboard")
	{
		dashboard.Static("/favicon_io", "dashboard/favicon_io")
		dashboard.Static("/build", "dashboard/build")
		dashboard.Static("/public", "dashboard/public")
		dashboard.GET("/", s.Dependencies.HTTPProvider.DashboardHandler())
		dashboard.GET("/:page", s.Dependencies.HTTPProvider.DashboardHandler())
	}
	return router
}

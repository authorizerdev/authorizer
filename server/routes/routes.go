package routes

import (
	"github.com/authorizerdev/authorizer/server/handlers"
	"github.com/authorizerdev/authorizer/server/middlewares"
	"github.com/gin-gonic/gin"
)

// InitRouter initializes gin router
func InitRouter() *gin.Engine {
	router := gin.Default()
	// router.Use(location.Default())
	router.Use(middlewares.GinContextToContextMiddleware())
	router.Use(middlewares.CORSMiddleware())

	router.GET("/", handlers.RootHandler())
	router.GET("/health", handlers.HealthHandler())
	router.POST("/graphql", handlers.GraphqlHandler())
	router.GET("/playground", handlers.PlaygroundHandler())
	router.GET("/oauth_login/:oauth_provider", handlers.OAuthLoginHandler())
	router.GET("/oauth_callback/:oauth_provider", handlers.OAuthCallbackHandler())
	router.GET("/verify_email", handlers.VerifyEmailHandler())

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
		dashboard.GET("/", handlers.DashboardHandler())
		dashboard.GET("/:page", handlers.DashboardHandler())
	}
	return router
}

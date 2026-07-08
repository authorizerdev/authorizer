package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/gen/openapi"
)

// spaBuildCacheMiddleware sets cache headers for SPA build assets:
//   - "index.js" / "main.css" (unhashed entry points the shell HTML loads
//     by name) → no-cache, so browsers always pick up new chunk references
//     after a deploy.
//   - everything else (content-hashed chunks, immutable assets) → long-lived
//     immutable cache, since a content change produces a new filename.
func spaBuildCacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		base := path.Base(c.Request.URL.Path)
		if base == "index.js" || base == "main.css" {
			c.Header("Cache-Control", "no-cache, must-revalidate")
		} else {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	}
}

// NewRouter creates new gin router
func (s *server) NewRouter() *gin.Engine {
	router := gin.New()
	// Restrict the set of proxies whose forwarded headers are honoured.
	// When TrustedProxies is empty/nil, gin trusts NO proxies and falls back
	// to RemoteAddr — preventing X-Forwarded-For spoofing for rate limiting,
	// audit logs, and CSRF same-origin comparisons.
	var trustedProxies []string
	if s.Dependencies.AppConfig != nil {
		trustedProxies = s.Dependencies.AppConfig.TrustedProxies
	}
	if err := router.SetTrustedProxies(trustedProxies); err != nil {
		s.Dependencies.Log.Warn().Err(err).Msg("failed to apply trusted proxies; falling back to gin defaults")
	}
	router.Use(gin.Recovery())

	router.Use(s.Dependencies.HTTPProvider.SecurityHeadersMiddleware())
	router.Use(s.Dependencies.HTTPProvider.LoggerMiddleware())
	router.Use(s.Dependencies.HTTPProvider.MetricsMiddleware())
	router.Use(s.Dependencies.HTTPProvider.ContextMiddleware())
	router.Use(s.Dependencies.HTTPProvider.CORSMiddleware())
	router.Use(s.Dependencies.HTTPProvider.RateLimitMiddleware())
	router.Use(s.Dependencies.HTTPProvider.CSRFMiddleware())
	router.Use(s.Dependencies.HTTPProvider.ClientCheckMiddleware())

	router.GET("/", s.Dependencies.HTTPProvider.RootHandler())
	router.GET("/health", s.Dependencies.HTTPProvider.HealthHandler())
	router.GET("/healthz", s.Dependencies.HTTPProvider.HealthHandler())
	router.GET("/readyz", s.Dependencies.HTTPProvider.ReadyHandler())
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
	router.POST("/logout", s.Dependencies.HTTPProvider.LogoutHandler())
	router.POST("/oauth/token", s.Dependencies.HTTPProvider.TokenHandler())
	router.POST("/oauth/revoke", s.Dependencies.HTTPProvider.RevokeRefreshTokenHandler())
	router.POST("/oauth/introspect", s.Dependencies.HTTPProvider.IntrospectHandler())

	// Inbound SCIM 2.0 (per-org user provisioning). Its own route group with a
	// bearer-token auth middleware; the org is derived only from the token, so
	// there is no org segment in the path (design §4.4 H6). CSRF is exempted for
	// /scim/v2/ in the CSRF middleware (bearer-authenticated, cookieless).
	if s.Dependencies.ScimHandler != nil {
		s.Dependencies.ScimHandler.Register(router.Group("/scim/v2"))
	}

	// gRPC-gateway REST surface at /v1/*. Mounted only when the gRPC
	// server is configured. Shares all gin middleware (CORS, security
	// headers, rate limit, logging) automatically since the route group
	// inherits them from `router.Use(...)` above.
	if s.gatewayHandler != nil {
		// The gateway's routes are registered with their full /v1/... path
		// (driven by google.api.http annotations). Mount it as a catch-all
		// under /v1 so gin matches the prefix and hands the full request
		// path to grpc-gateway untouched.
		gw := gin.WrapH(s.gatewayHandler)
		router.Any("/v1/*path", gw)

		// OpenAPI spec — generated alongside the gRPC stubs by buf and
		// embedded into the binary (so it works regardless of cwd: tests,
		// containers, etc.). Path is intentionally separate from the
		// gateway mux so it doesn't fight a /v1/openapi.json gateway route.
		router.GET("/openapi.json", func(c *gin.Context) {
			c.Data(http.StatusOK, "application/json", openapi.Spec())
		})
	}

	// Set up template functions for JSON encoding.
	// Escape </script> and <!-- to prevent script injection in <script> blocks.
	router.SetFuncMap(template.FuncMap{
		"json": func(v interface{}) template.JS {
			a, _ := json.Marshal(v)
			s := string(a)
			s = strings.ReplaceAll(s, "</", `<\/`)
			s = strings.ReplaceAll(s, "<!--", `<\!--`)
			return template.JS(s)
		},
	})
	router.LoadHTMLGlob("web/templates/*")
	// // login page app related routes.
	app := router.Group("/app")
	{
		app.Static("/favicon_io", "web/app/favicon_io")
		appBuild := app.Group("/build")
		appBuild.Use(spaBuildCacheMiddleware())
		appBuild.Static("", "web/app/build")
		app.GET("/", s.Dependencies.HTTPProvider.AppHandler())
		app.GET("/:page", s.Dependencies.HTTPProvider.AppHandler())
	}

	// // dashboard related routes
	dashboard := router.Group("/dashboard")
	{
		dashboard.Static("/favicon_io", "web/dashboard/favicon_io")
		dashboardBuild := dashboard.Group("/build")
		dashboardBuild.Use(spaBuildCacheMiddleware())
		dashboardBuild.Static("", "web/dashboard/build")
		dashboard.Static("/public", "web/dashboard/public")
		dashboard.GET("/", s.Dependencies.HTTPProvider.DashboardHandler())
		dashboard.GET("/:page", s.Dependencies.HTTPProvider.DashboardHandler())
	}

	// SPA fallback: any unmatched GET inside /app/ or /dashboard/ serves the
	// SPA shell so deep links and browser refresh on multi-segment routes
	// (e.g. /dashboard/authorization/resources) don't return 404. Static
	// routes (/build, /favicon_io, /public) and the explicit /, /:page
	// handlers above take precedence; this only catches the multi-segment
	// gap. Non-GET methods and other paths fall through to gin's default
	// 404 handler.
	dashboardHandler := s.Dependencies.HTTPProvider.DashboardHandler()
	appHandler := s.Dependencies.HTTPProvider.AppHandler()
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != "GET" {
			c.AbortWithStatus(404)
			return
		}
		path := c.Request.URL.Path
		switch {
		case strings.HasPrefix(path, "/dashboard/"):
			dashboardHandler(c)
		case strings.HasPrefix(path, "/app/"):
			appHandler(c)
		default:
			c.AbortWithStatus(404)
		}
	})
	return router
}

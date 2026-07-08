package http_handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/events"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/oauth"
	"github.com/authorizerdev/authorizer/internal/rate_limit"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/service/clientauth"
	"github.com/authorizerdev/authorizer/internal/sms"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/token"
)

// Dependencies for a graphql provider
type Dependencies struct {
	Log *zerolog.Logger

	// Providers for various services
	// AuditProvider is used to log audit events
	AuditProvider audit.Provider
	// AuthenticatorProvider is used to register authenticators like totp (Google Authenticator)
	AuthenticatorProvider authenticators.Provider
	// EmailProvider is used to send emails
	EmailProvider email.Provider
	// EventsProvider is used to register events
	EventsProvider events.Provider
	// MemoryStoreProvider is used to store data in memory
	MemoryStoreProvider memory_store.Provider
	// SMSProvider is used to send SMS
	SMSProvider sms.Provider
	// StorageProvider is used to register storage like database
	StorageProvider storage.Provider
	// TokenProvider is used to generate tokens
	TokenProvider token.Provider
	// OAuthProvider is used to register oauth providers
	OAuthProvider oauth.Provider
	// RateLimitProvider is used for per-IP rate limiting
	RateLimitProvider rate_limit.Provider
	// ServiceProvider hosts the transport-agnostic public-API operations.
	// Migrated GraphQL resolvers delegate here.
	ServiceProvider service.Provider
	// AuthzEngine is the fine-grained authorization (FGA) engine.
	// It is nil unless an FGA store is configured (--fga-store).
	AuthzEngine engine.AuthorizationEngine
}

// New constructs a new http provider with given arguments
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	// TODO - Add any validation here for config and dependencies
	g := &httpProvider{
		Config:       cfg,
		Dependencies: *deps,
		// Shared client-authentication resolver (RFC 6749 §2.3). Built from the
		// same storage + config the handlers already hold, so no extra wiring is
		// needed at construction sites.
		clientAuthProvider: clientauth.New(cfg, &clientauth.Dependencies{
			Log:                 deps.Log,
			StorageProvider:     deps.StorageProvider,
			MemoryStoreProvider: deps.MemoryStoreProvider,
		}),
	}
	return g, nil
}

// httpProvider is the struct that provides resolver functions for http routes.
type httpProvider struct {
	*config.Config
	Dependencies
	// clientAuthProvider resolves and authenticates the OAuth client presented at
	// the token endpoint.
	clientAuthProvider clientauth.Provider
}

// Ensure interface is implemented
var _ Provider = &httpProvider{}

// Provider is the interface that provides the methods to interact with the http handlers.
type Provider interface {
	// AppHandler is the main handler that handels all the app requests
	AppHandler() gin.HandlerFunc
	// AuthorizeHandler is the main handler that handels all the authorize requests
	AuthorizeHandler() gin.HandlerFunc
	// DashboardHandler is the main handler that handels all the dashboard requests
	DashboardHandler() gin.HandlerFunc
	// GraphqlHandler is the main handler that handels all the graphql requests
	GraphqlHandler() gin.HandlerFunc
	// HealthHandler is the handler for the /healthz liveness probe
	HealthHandler() gin.HandlerFunc
	// ReadyHandler is the handler for the /readyz readiness probe
	ReadyHandler() gin.HandlerFunc
	// IntrospectHandler is the main handler for RFC 7662 Token Introspection
	IntrospectHandler() gin.HandlerFunc
	// JWKsHandler is the main handler that handels all the jwks requests
	JWKsHandler() gin.HandlerFunc
	// LogoutHandler is the main handler that handels all the logout requests
	LogoutHandler() gin.HandlerFunc
	// OAuthCallbackHandler is the main handler that handels all the oauth callback requests
	OAuthCallbackHandler() gin.HandlerFunc
	// OAuthLoginHandler is the main handler that handels all the oauth login requests
	OAuthLoginHandler() gin.HandlerFunc
	// SSOLoginHandler starts a per-org enterprise OIDC SSO broker login.
	SSOLoginHandler() gin.HandlerFunc
	// SSOCallbackHandler completes a per-org enterprise OIDC SSO broker login.
	SSOCallbackHandler() gin.HandlerFunc
	// SAMLMetadataHandler serves a per-org SP SAML metadata document.
	SAMLMetadataHandler() gin.HandlerFunc
	// SAMLLoginHandler starts a per-org SP-initiated SAML SSO login.
	SAMLLoginHandler() gin.HandlerFunc
	// SAMLACSHandler consumes a per-org SAML assertion (Assertion Consumer Service).
	SAMLACSHandler() gin.HandlerFunc
	// OpenIDConfigurationHandler is the main handler that handels all the openid configuration requests
	OpenIDConfigurationHandler() gin.HandlerFunc
	// PlaygroundHandler is the main handler that handels all the playground requests
	PlaygroundHandler() gin.HandlerFunc
	// RevokeRefreshTokenHandler is the main handler that handels all the revoke refresh token requests
	RevokeRefreshTokenHandler() gin.HandlerFunc
	// RootHandler is the main handler that handels all the root requests
	RootHandler() gin.HandlerFunc
	// TokenHandler is the main handler that handels all the token requests
	TokenHandler() gin.HandlerFunc
	// UserInfoHandler is the main handler that handels all the user info requests
	UserInfoHandler() gin.HandlerFunc
	// VerifyEmailHandler is the main handler that handels all the verify email requests
	VerifyEmailHandler() gin.HandlerFunc

	// ClientCheckMiddleware is the middleware that checks if the client is valid
	ClientCheckMiddleware() gin.HandlerFunc
	// ContextMiddleware is the middleware that adds the context to the request
	ContextMiddleware() gin.HandlerFunc
	// CORSMiddleware is the middleware that adds the cors headers to the response
	CORSMiddleware() gin.HandlerFunc
	// CSRFMiddleware protects against CSRF on state-changing requests
	CSRFMiddleware() gin.HandlerFunc
	// LoggerMiddleware is the middleware that logs the request
	LoggerMiddleware() gin.HandlerFunc
	// RateLimitMiddleware is the middleware that rate limits requests per IP
	RateLimitMiddleware() gin.HandlerFunc
	// MetricsMiddleware records HTTP request count and duration for prometheus.
	MetricsMiddleware() gin.HandlerFunc
	// MetricsHandler serves the Prometheus metrics scrape endpoint.
	MetricsHandler() gin.HandlerFunc
	// SecurityHeadersMiddleware sets standard security headers on every response.
	SecurityHeadersMiddleware() gin.HandlerFunc
}

package http_handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/events"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/oauth"
	"github.com/authorizerdev/authorizer/internal/sms"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/token"
)

// Dependencies for a graphql provider
type Dependencies struct {
	Log *zerolog.Logger

	// Providers for various services
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
}

// New constructs a new http provider with given arguments
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	// TODO - Add any validation here for config and dependencies
	g := &httpProvider{
		Config:       cfg,
		Dependencies: *deps,
	}
	return g, nil
}

// httpProvider is the struct that provides resolver functions for http routes.
type httpProvider struct {
	*config.Config
	Dependencies
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
	// HealthHandler is the main handler that handels all the health requests
	HealthHandler() gin.HandlerFunc
	// JWKsHandler is the main handler that handels all the jwks requests
	JWKsHandler() gin.HandlerFunc
	// LogoutHandler is the main handler that handels all the logout requests
	LogoutHandler() gin.HandlerFunc
	// OAuthCallbackHandler is the main handler that handels all the oauth callback requests
	OAuthCallbackHandler() gin.HandlerFunc
	// OAuthLoginHandler is the main handler that handels all the oauth login requests
	OAuthLoginHandler() gin.HandlerFunc
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
	// LoggerMiddleware is the middleware that logs the request
	LoggerMiddleware() gin.HandlerFunc
}

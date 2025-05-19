package oauth

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"github.com/authorizerdev/authorizer/internal/config"
)

// oauthProvider is a struct that contains reference to the config and dependencies
// It is used to initialize the OAuth providers
type oauthProvider struct {
	*config.Config
}

// Ensure interface is implemented
var _ Provider = &oauthProvider{}

// Provider is the interface that provides the methods to interact with the oauth providers.
type Provider interface {
	// GetOAuthConfig returns the OAuth config for the given provider
	GetOAuthConfig(ctx *gin.Context, provider string) (*oauth2.Config, error)
}

// New constructs a new graphql provider with given arguments
func New(cfg *config.Config) (Provider, error) {
	g := &oauthProvider{
		Config: cfg,
	}
	return g, nil
}

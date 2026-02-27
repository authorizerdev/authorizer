package oauth

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"

	"github.com/authorizerdev/authorizer/internal/config"
)

// Dependencies struct for memory store provider
type Dependencies struct {
	Log *zerolog.Logger
}

// oauthProvider is a struct that contains reference to the config and dependencies
// It is used to initialize the OAuth providers
type oauthProvider struct {
	*config.Config
	*Dependencies
}

// Ensure interface is implemented
var _ Provider = &oauthProvider{}

// Provider is the interface that provides the methods to interact with the oauth providers.
type Provider interface {
	// GetOAuthConfig returns the OAuth config for the given provider
	GetOAuthConfig(ctx *gin.Context, provider string) (*oauth2.Config, error)
}

// New constructs a new graphql provider with given arguments
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	g := &oauthProvider{
		Config:       cfg,
		Dependencies: deps,
	}
	return g, nil
}

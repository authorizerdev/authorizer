package server

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/graphql"
	"github.com/authorizerdev/authorizer/internal/http_handlers"
)

// Configuration of a server.
type Config struct {
	// Host address to accept requests on
	Host string
	// Port number to serve HTTP requests on
	HTTPPort int
	// Port number to serve Metrics requests on
	MetricsPort int
}

// Dependencies for a server
type Dependencies struct {
	Log             *zerolog.Logger
	GraphQLProvider graphql.Provider
	HTTPProvider    http_handlers.Provider
}

// New constructs a new server with given arguments
func New(cfg *Config, deps *Dependencies) (*server, error) {
	s := &server{
		Config:       cfg,
		Dependencies: deps,
	}
	return s, nil
}

// Network server
type server struct {
	Config       *Config
	Dependencies *Dependencies
}

// Run the server until the given context is canceled
func (s *server) Run(ctx context.Context) error {
	// Create new router
	ginRouter := s.NewRouter()
	// Start the server
	go func() {
		s.Dependencies.Log.Info().Str("host", s.Config.Host).Int("port", s.Config.HTTPPort).Msg("Starting HTTP server")
		err := ginRouter.Run(s.Config.Host + ":" + fmt.Sprintf("%d", s.Config.HTTPPort))
		if err != nil {
			s.Dependencies.Log.Error().Err(err).Msg("HTTP server failed")
		}
	}()
	// Wait until context closed
	<-ctx.Done()
	return nil
}

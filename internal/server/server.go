package server

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/router"
	"github.com/authorizerdev/authorizer/internal/service"
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
	Log     *zerolog.Logger
	Service service.Service
}

// New constructs a new server with given arguments
func New(cfg *Config, deps Dependencies) (*server, error) {
	s := &server{
		Config:       *cfg,
		Dependencies: deps,
	}
	return s, nil
}

// Network server
type server struct {
	Config
	Dependencies
}

// Run the server until the given context is canceled
func (s *server) Run(ctx context.Context) error {
	// Create new router
	ginRouter := router.NewRouter()
	// Start the server
	go func() {
		s.Log.Info().Str("host", s.Host).Int("port", s.HTTPPort).Msg("Starting HTTP server")
		err := ginRouter.Run(s.Host + ":" + fmt.Sprintf("%d", s.HTTPPort))
		if err != nil {
			s.Log.Error().Err(err).Msg("HTTP server failed")
		}
	}()
	// Wait until context closed
	<-ctx.Done()
	return nil
}

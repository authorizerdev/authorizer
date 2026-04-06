package server

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	// MetricsHost is the bind address for the dedicated /metrics listener.
	MetricsHost string
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

// Run the server until the given context is canceled.
// The main HTTP server (Gin) and the Prometheus /metrics server always run as separate listeners.
func (s *server) Run(ctx context.Context) error {
	ginRouter := s.NewRouter()
	httpAddr := net.JoinHostPort(s.Config.Host, strconv.Itoa(s.Config.HTTPPort))
	go func() {
		s.Dependencies.Log.Info().Str("addr", httpAddr).Msg("Starting HTTP server")
		if err := ginRouter.Run(httpAddr); err != nil {
			s.Dependencies.Log.Error().Err(err).Msg("HTTP server failed")
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	metricsAddr := net.JoinHostPort(s.Config.MetricsHost, strconv.Itoa(s.Config.MetricsPort))
	metricsSrv := &http.Server{
		Addr:    metricsAddr,
		Handler: mux,
	}
	go func() {
		s.Dependencies.Log.Info().Str("addr", metricsAddr).Msg("Starting metrics server")
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Dependencies.Log.Error().Err(err).Msg("Metrics server failed")
		}
	}()
	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = metricsSrv.Shutdown(shCtx)
	}()

	<-ctx.Done()
	return nil
}

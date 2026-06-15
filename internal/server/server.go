package server

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/gateway"
	"github.com/authorizerdev/authorizer/internal/graphql"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	"github.com/authorizerdev/authorizer/internal/http_handlers"
)

// Config holds the configuration of a server.
type Config struct {
	// Host address to accept requests on
	Host string
	// Port number to serve HTTP requests on
	HTTPPort int
	// Port number to serve Metrics requests on
	MetricsPort int
	// MetricsHost is the bind address for the dedicated /metrics listener.
	MetricsHost string
	// GRPCPort is the port the gRPC server listens on.
	GRPCPort int
}

// Dependencies for a server
type Dependencies struct {
	Log             *zerolog.Logger
	AppConfig       *config.Config
	GraphQLProvider graphql.Provider
	HTTPProvider    http_handlers.Provider
	// GRPCServer is the configured (but not yet listening) gRPC server.
	// nil disables both the gRPC listener and the REST `/v1/*` gateway.
	GRPCServer *grpcsrv.Server
	// gatewayHandler / gatewayCleanup are built lazily inside Run when
	// GRPCServer is non-nil. Stored on the struct only to satisfy the
	// existing pattern of cleanup at Shutdown time.
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
	Config         *Config
	Dependencies   *Dependencies
	gatewayHandler http.Handler
}

// Run the server until the given context is canceled.
// The main HTTP server (Gin), the Prometheus /metrics server, and the gRPC
// server (when configured) all run as separate listeners.
func (s *server) Run(ctx context.Context) error {
	// Build the REST gateway BEFORE the router so it can be mounted at
	// /v1/*. The gateway dials the gRPC server in-process via bufconn —
	// no extra port hop, no TLS plumbing.
	var gatewayCleanup func()
	if s.Dependencies.GRPCServer != nil {
		h, cleanup, err := gateway.Handler(ctx, s.Dependencies.GRPCServer.GRPCServer())
		if err != nil {
			return err
		}
		s.gatewayHandler = h
		gatewayCleanup = cleanup
		defer gatewayCleanup()
	}

	ginRouter := s.NewRouter()
	httpAddr := net.JoinHostPort(s.Config.Host, strconv.Itoa(s.Config.HTTPPort))

	// Build the main HTTP server explicitly with conservative timeouts to
	// defend against slowloris and other slow-client DoS. ReadHeaderTimeout
	// is the most important — it bounds time to receive request headers,
	// which is what slowloris exploits.
	httpSrv := &http.Server{
		Addr:              httpAddr,
		Handler:           ginRouter,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}
	go func() {
		s.Dependencies.Log.Info().Str("addr", httpAddr).Msg("Starting HTTP server")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Dependencies.Log.Error().Err(err).Msg("HTTP server failed")
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	metricsAddr := net.JoinHostPort(s.Config.MetricsHost, strconv.Itoa(s.Config.MetricsPort))
	metricsSrv := &http.Server{
		Addr:              metricsAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	go func() {
		s.Dependencies.Log.Info().Str("addr", metricsAddr).Msg("Starting metrics server")
		if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Dependencies.Log.Error().Err(err).Msg("Metrics server failed")
		}
	}()

	// gRPC listener — runs alongside HTTP and metrics. Cancelled context
	// triggers graceful shutdown.
	if s.Dependencies.GRPCServer != nil {
		grpcAddr := net.JoinHostPort(s.Config.Host, strconv.Itoa(s.Config.GRPCPort))
		go func() {
			if err := s.startGRPC(ctx, grpcAddr); err != nil {
				s.Dependencies.Log.Error().Err(err).Msg("gRPC server failed")
			}
		}()
	}

	// Graceful shutdown for BOTH servers (previously only metrics was
	// shut down gracefully — the main HTTP server was killed mid-flight,
	// dropping in-progress responses).
	<-ctx.Done()
	s.Dependencies.Log.Info().Msg("Shutdown signal received, draining connections")
	shCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shCtx); err != nil {
		s.Dependencies.Log.Error().Err(err).Msg("HTTP server graceful shutdown failed")
	}
	if err := metricsSrv.Shutdown(shCtx); err != nil {
		s.Dependencies.Log.Error().Err(err).Msg("Metrics server graceful shutdown failed")
	}
	return nil
}

// startGRPC runs the gRPC server until ctx is cancelled. Delegates the TCP
// listen + graceful-stop dance to grpcsrv.Server.Run; this wrapper exists
// only to keep all listener startup colocated in server.Run.
func (s *server) startGRPC(ctx context.Context, addr string) error {
	// grpcsrv.Server was constructed with its own addr; honour that.
	// (When we add CLI-driven addr override, this is the place.)
	_ = addr
	return s.Dependencies.GRPCServer.Run(ctx)
}

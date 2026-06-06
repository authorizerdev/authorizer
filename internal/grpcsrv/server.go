// Package grpcsrv builds and runs the Authorizer gRPC server. It registers
// every public-API service (real or stubbed), enables reflection, exposes
// the standard gRPC health checking protocol, and applies the shared
// interceptor chain.
package grpcsrv

import (
	"context"
	"net"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/grpcsrv/handlers"
	"github.com/authorizerdev/authorizer/internal/grpcsrv/interceptors"
	"github.com/authorizerdev/authorizer/internal/service"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// Dependencies is the minimum set the gRPC server needs.
type Dependencies struct {
	Log             *zerolog.Logger
	Config          *config.Config
	ServiceProvider service.Provider
}

// Server wraps a *grpc.Server plus its listener address.
type Server struct {
	deps   *Dependencies
	addr   string
	srv    *grpc.Server
	health *health.Server
}

// New constructs a configured gRPC server. The server is not yet listening;
// call Run to start serving.
func New(addr string, deps *Dependencies) (*Server, error) {
	validate, err := interceptors.Validate()
	if err != nil {
		return nil, err
	}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.Recovery(deps.Log),
			interceptors.Logging(deps.Log),
			validate,
			// Innermost: wraps the handler directly so it can translate typed
			// service.Error values into proper gRPC status codes. Must stay
			// last — see interceptors.ErrorMap docs.
			interceptors.ErrorMap(),
		),
	)

	// Register the single AuthorizerService. AuthorizerHandler embeds
	// UnimplementedAuthorizerServiceServer, so any RPC whose method has
	// not yet been migrated returns codes.Unimplemented. Migrated methods
	// (today: Meta) override the unimplemented stubs.
	authorizerv1.RegisterAuthorizerServiceServer(srv, &handlers.AuthorizerHandler{Service: deps.ServiceProvider})

	// gRPC health checking protocol (used by k8s grpc-probe and similar).
	hs := health.NewServer()
	hs.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(srv, hs)

	// Reflection is gated on a config flag so prod deployments can lock it
	// off, but defaults on for dev/test parity with the playground.
	if deps.Config.EnableGRPCReflection {
		reflection.Register(srv)
	}

	return &Server{
		deps:   deps,
		addr:   addr,
		srv:    srv,
		health: hs,
	}, nil
}

// GRPCServer exposes the underlying *grpc.Server. Used by the in-process
// REST gateway mount to dial via bufconn during tests.
func (s *Server) GRPCServer() *grpc.Server { return s.srv }

// Run starts the listener and blocks until ctx is cancelled or Serve errors.
// On context cancellation, the server is gracefully stopped (existing RPCs
// finish, no new ones accepted).
func (s *Server) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.deps.Log.Info().Str("addr", s.addr).Msg("Starting gRPC server")

	errCh := make(chan error, 1)
	go func() { errCh <- s.srv.Serve(lis) }()

	select {
	case <-ctx.Done():
		s.deps.Log.Info().Msg("gRPC shutdown signal received, draining")
		s.health.Shutdown()
		s.srv.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

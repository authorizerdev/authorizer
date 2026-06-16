package integration_tests

import (
	"context"
	"net"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	"github.com/authorizerdev/authorizer/internal/service"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// TestGRPCMeta exercises AuthorizerService.Meta end-to-end over a bufconn
// in-process gRPC channel. Validates the consolidated single-service
// design: proto → handler → service.Meta → response projection.
func TestGRPCMeta(t *testing.T) {
	cfg := getTestConfig()
	cfg.ClientID = "test-client"

	log := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	svc, err := service.New(cfg, &service.Dependencies{Log: &log})
	require.NoError(t, err)

	srv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             &log,
		Config:          cfg,
		ServiceProvider: svc,
	})
	require.NoError(t, err)

	lis := bufconn.Listen(1 << 20)
	t.Cleanup(func() { _ = lis.Close() })
	go func() { _ = srv.GRPCServer().Serve(lis) }()
	t.Cleanup(srv.GRPCServer().GracefulStop)

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	client := authorizerv1.NewAuthorizerServiceClient(conn)
	resp, err := client.Meta(context.Background(), &authorizerv1.MetaRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "test-client", resp.ClientId)
	require.NotEmpty(t, resp.Version)
}

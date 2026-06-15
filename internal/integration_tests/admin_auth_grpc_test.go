package integration_tests

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// newAdminClient boots an in-process gRPC server backed by the same fully-wired
// service provider the GraphQL path uses (via initTestSetup), and returns an
// AuthorizerAdminService client plus the test config. The admin service is
// registered on the same single server as the public AuthorizerService.
func newAdminClient(t *testing.T) (authorizerv1.AuthorizerAdminServiceClient, *config.Config) {
	t.Helper()
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	srv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             ts.Logger,
		Config:          cfg,
		ServiceProvider: ts.ServiceProvider,
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

	return authorizerv1.NewAuthorizerAdminServiceClient(conn), cfg
}

// adminCtx returns a context carrying the admin-secret metadata that
// transport.MetaFromGRPC forwards to the super-admin auth check.
func adminCtx(secret string) context.Context {
	return metadata.NewOutgoingContext(context.Background(),
		metadata.Pairs("x-authorizer-admin-secret", secret))
}

// TestAdminMetaGRPC exercises AuthorizerAdminService.AdminMeta over gRPC,
// asserting the fail-closed contract (no secret → Unauthenticated) and the
// happy path.
func TestAdminMetaGRPC(t *testing.T) {
	client, cfg := newAdminClient(t)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.AdminMeta(context.Background(), &authorizerv1.AdminMetaRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns roles with admin secret", func(t *testing.T) {
		resp, err := client.AdminMeta(adminCtx(cfg.AdminSecret), &authorizerv1.AdminMetaRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp.AdminMeta)
		require.Equal(t, cfg.Roles, resp.AdminMeta.Roles)
		require.Equal(t, cfg.DefaultRoles, resp.AdminMeta.DefaultRoles)
	})
}

// TestAdminLoginGRPC exercises AuthorizerAdminService.AdminLogin over gRPC.
// Login is the only admin RPC that does not require an existing session.
func TestAdminLoginGRPC(t *testing.T) {
	client, cfg := newAdminClient(t)

	t.Run("invalid secret is rejected", func(t *testing.T) {
		_, err := client.AdminLogin(context.Background(), &authorizerv1.AdminLoginRequest{
			AdminSecret: "wrong-secret",
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("valid secret logs in", func(t *testing.T) {
		resp, err := client.AdminLogin(context.Background(), &authorizerv1.AdminLoginRequest{
			AdminSecret: cfg.AdminSecret,
		})
		require.NoError(t, err)
		require.Equal(t, "admin logged in successfully", resp.Message)
	})
}

// TestAdminSessionAndLogoutGRPC exercises the cookie-bearing admin RPCs over
// gRPC, asserting both the fail-closed contract and the happy path.
func TestAdminSessionAndLogoutGRPC(t *testing.T) {
	client, cfg := newAdminClient(t)

	t.Run("session refresh requires admin", func(t *testing.T) {
		_, err := client.AdminSession(context.Background(), &authorizerv1.AdminSessionRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))

		resp, err := client.AdminSession(adminCtx(cfg.AdminSecret), &authorizerv1.AdminSessionRequest{})
		require.NoError(t, err)
		require.Equal(t, "admin session refreshed successfully", resp.Message)
	})

	t.Run("logout requires admin", func(t *testing.T) {
		_, err := client.AdminLogout(context.Background(), &authorizerv1.AdminLogoutRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))

		resp, err := client.AdminLogout(adminCtx(cfg.AdminSecret), &authorizerv1.AdminLogoutRequest{})
		require.NoError(t, err)
		require.Equal(t, "admin logged out successfully", resp.Message)
	})
}

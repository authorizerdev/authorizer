package integration_tests

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

// newPublicClient boots an in-process gRPC server backed by the same fully-wired
// service provider the GraphQL path uses (via initTestSetup) and returns an
// AuthorizerService client plus the test config. This is the public-API
// counterpart to newAdminClient.
func newPublicClient(t *testing.T) (authorizerv1.AuthorizerServiceClient, *config.Config) {
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

	return authorizerv1.NewAuthorizerServiceClient(conn), cfg
}

// bearerCtx returns a context carrying a Bearer access token that
// transport.MetaFromGRPC forwards to the session/access-token auth check.
func bearerCtx(token string) context.Context {
	return metadata.NewOutgoingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer "+token))
}

// TestLoginGRPC exercises the migrated Login RPC end-to-end over gRPC: sign up a
// user, then prove the credential checks and the happy path all run through the
// shared service layer (this was the original "Login not implemented for gRPC"
// gap).
func TestLoginGRPC(t *testing.T) {
	c, _ := newPublicClient(t)
	ctx := context.Background()

	email := "grpc_login_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	_, err := c.Signup(ctx, &authorizerv1.SignupRequest{
		Email:           email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	t.Run("invalid password is rejected (not Unimplemented)", func(t *testing.T) {
		_, err := c.Login(ctx, &authorizerv1.LoginRequest{Email: email, Password: "WrongPassword@123"})
		require.Error(t, err)
		assert.NotEqual(t, codes.Unimplemented, status.Code(err))
	})

	t.Run("empty credentials are rejected with InvalidArgument", func(t *testing.T) {
		_, err := c.Login(ctx, &authorizerv1.LoginRequest{Password: password})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("valid credentials log in and return an access token", func(t *testing.T) {
		resp, err := c.Login(ctx, &authorizerv1.LoginRequest{Email: email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.AccessToken)
		require.NotNil(t, resp.User)
		assert.Equal(t, email, resp.User.Email)
	})
}

// TestUpdateProfileGRPC proves the auth-required UpdateProfile RPC enforces
// authentication and, with a valid bearer token from Login, applies the update
// through the shared service layer.
func TestUpdateProfileGRPC(t *testing.T) {
	c, _ := newPublicClient(t)
	ctx := context.Background()

	email := "grpc_updprofile_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	_, err := c.Signup(ctx, &authorizerv1.SignupRequest{
		Email:           email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	t.Run("fails without auth", func(t *testing.T) {
		_, err := c.UpdateProfile(ctx, &authorizerv1.UpdateProfileRequest{GivenName: "Nobody"})
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("succeeds with a bearer token", func(t *testing.T) {
		loginResp, err := c.Login(ctx, &authorizerv1.LoginRequest{Email: email, Password: password})
		require.NoError(t, err)
		require.NotNil(t, loginResp)
		require.NotEmpty(t, loginResp.AccessToken)

		resp, err := c.UpdateProfile(bearerCtx(loginResp.AccessToken), &authorizerv1.UpdateProfileRequest{
			GivenName: "Updated",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Message)
	})
}

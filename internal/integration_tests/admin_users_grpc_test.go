package integration_tests

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// newAdminClientWithSetup boots the same in-process admin gRPC server as
// newAdminClient but also returns the *testSetup so a test can seed storage
// deterministically (e.g. an existing user for User/UpdateUser/DeleteUser).
func newAdminClientWithSetup(t *testing.T) (authorizerv1.AuthorizerAdminServiceClient, *testSetup) {
	t.Helper()
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	srv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             ts.Logger,
		Config:          cfg,
		ServiceProvider: ts.ServiceProvider,
		TokenProvider:   ts.TokenProvider,
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

	return authorizerv1.NewAuthorizerAdminServiceClient(conn), ts
}

// seedUser inserts a deterministic basic-auth user directly via storage and
// returns its id and email. Used by the admin user RPC tests.
func seedUser(t *testing.T, ts *testSetup) (id, email string) {
	t.Helper()
	id = uuid.New().String()
	email = "admin-users-grpc-" + id + "@authorizer.test"
	now := int64(1)
	_, err := ts.StorageProvider.AddUser(context.Background(), &schemas.User{
		ID:              id,
		Email:           refs.NewStringRef(email),
		SignupMethods:   constants.AuthRecipeMethodBasicAuth,
		Roles:           "user",
		EmailVerifiedAt: &now,
	})
	require.NoError(t, err)
	return id, email
}

// TestAdminUsersGRPC exercises AuthorizerAdminService.Users over gRPC: the
// fail-closed contract (no secret → Unauthenticated) and the happy path with a
// seeded user present in the returned page.
func TestAdminUsersGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	_, email := seedUser(t, ts)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.Users(context.Background(), &authorizerv1.UsersRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns users with admin secret", func(t *testing.T) {
		resp, err := client.Users(adminCtx(cfg.AdminSecret), &authorizerv1.UsersRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp.Pagination)
		var found bool
		for _, u := range resp.Users {
			if u.Email == email {
				found = true
				break
			}
		}
		require.True(t, found, "seeded user should appear in the users page")
	})
}

// TestAdminUserGRPC exercises AuthorizerAdminService.User over gRPC for both
// the id and email lookup paths plus the fail-closed contract.
func TestAdminUserGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	id, email := seedUser(t, ts)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.User(context.Background(), &authorizerv1.UserRequest{Id: id})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("get by id", func(t *testing.T) {
		resp, err := client.User(adminCtx(cfg.AdminSecret), &authorizerv1.UserRequest{Id: id})
		require.NoError(t, err)
		require.NotNil(t, resp.User)
		require.Equal(t, id, resp.User.Id)
		require.Equal(t, email, resp.User.Email)
	})

	t.Run("get by email", func(t *testing.T) {
		resp, err := client.User(adminCtx(cfg.AdminSecret), &authorizerv1.UserRequest{Email: email})
		require.NoError(t, err)
		require.NotNil(t, resp.User)
		require.Equal(t, id, resp.User.Id)
	})

	t.Run("missing params is an error", func(t *testing.T) {
		_, err := client.User(adminCtx(cfg.AdminSecret), &authorizerv1.UserRequest{})
		require.Error(t, err)
	})
}

// TestAdminUpdateUserGRPC exercises AuthorizerAdminService.UpdateUser over gRPC:
// the fail-closed contract and a happy-path profile update applied to a seeded
// user.
func TestAdminUpdateUserGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	id, _ := seedUser(t, ts)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.UpdateUser(context.Background(), &authorizerv1.UpdateUserRequest{
			Id:        id,
			GivenName: refs.NewStringRef("Ada"),
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("updates given name", func(t *testing.T) {
		resp, err := client.UpdateUser(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateUserRequest{
			Id:        id,
			GivenName: refs.NewStringRef("Ada"),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.User)
		require.Equal(t, "Ada", resp.User.GivenName)
	})

	t.Run("no update params is an error", func(t *testing.T) {
		_, err := client.UpdateUser(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateUserRequest{Id: id})
		require.Error(t, err)
	})
}

// TestAdminDeleteUserGRPC exercises AuthorizerAdminService.DeleteUser over gRPC:
// the fail-closed contract and a happy-path delete of a seeded user.
func TestAdminDeleteUserGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	_, email := seedUser(t, ts)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.DeleteUser(context.Background(), &authorizerv1.DeleteUserRequest{Email: email})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("deletes user", func(t *testing.T) {
		resp, err := client.DeleteUser(adminCtx(cfg.AdminSecret), &authorizerv1.DeleteUserRequest{Email: email})
		require.NoError(t, err)
		require.Equal(t, "user deleted successfully", resp.Message)
	})

	t.Run("deleting unknown user is an error", func(t *testing.T) {
		_, err := client.DeleteUser(adminCtx(cfg.AdminSecret), &authorizerv1.DeleteUserRequest{
			Email: "does-not-exist-" + uuid.New().String() + "@authorizer.test",
		})
		require.Error(t, err)
	})
}

// TestAdminVerificationRequestsGRPC exercises
// AuthorizerAdminService.VerificationRequests over gRPC: the fail-closed
// contract and the happy path (empty list is valid).
func TestAdminVerificationRequestsGRPC(t *testing.T) {
	client, cfg := newAdminClient(t)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.VerificationRequests(context.Background(), &authorizerv1.VerificationRequestsRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns verification requests with admin secret", func(t *testing.T) {
		resp, err := client.VerificationRequests(adminCtx(cfg.AdminSecret), &authorizerv1.VerificationRequestsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp.Pagination)
	})
}

package integration_tests

import (
	"context"
	"net"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	"github.com/authorizerdev/authorizer/internal/service"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// bootGRPCBufconn builds a gRPC server identical to the production one,
// served over an in-process bufconn. Returns a dialed *grpc.ClientConn the
// test uses to issue real RPCs.
func bootGRPCBufconn(t *testing.T) *grpc.ClientConn {
	t.Helper()
	cfg := getTestConfig()
	cfg.ClientID = "test-client"
	log := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	svc, err := service.New(cfg, &service.Dependencies{Log: &log})
	require.NoError(t, err)
	srv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{Log: &log, Config: cfg, ServiceProvider: svc})
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
	return conn
}

// TestAuthorizerServiceMigratedRPCsAreImplemented is the positive contract for
// the 10 auth RPCs that were migrated from the GraphQL resolvers into the
// shared service layer (Login, MagicLinkLogin, VerifyEmail, ResendVerifyEmail,
// VerifyOtp, ResendOtp, ForgotPassword, ResetPassword, UpdateProfile,
// DeactivateAccount). With the rest of the surface this brings all 20
// AuthorizerService RPCs live. Each is invoked with minimal/invalid input —
// the exact status varies (InvalidArgument / Unauthenticated /
// FailedPrecondition / Internal, or a deliberately-generic success on the
// account-enumeration-safe paths) but it MUST NEVER be codes.Unimplemented.
func TestAuthorizerServiceMigratedRPCsAreImplemented(t *testing.T) {
	conn := bootGRPCBufconn(t)
	ctx := context.Background()
	c := authorizerv1.NewAuthorizerServiceClient(conn)

	type call func(context.Context) error
	cases := map[string]call{
		"Login": func(c0 context.Context) error {
			_, err := c.Login(c0, &authorizerv1.LoginRequest{Email: "x@example.com", Password: "p"})
			return err
		},
		"MagicLinkLogin": func(c0 context.Context) error {
			_, err := c.MagicLinkLogin(c0, &authorizerv1.MagicLinkLoginRequest{Email: "x@example.com"})
			return err
		},
		"VerifyEmail": func(c0 context.Context) error {
			_, err := c.VerifyEmail(c0, &authorizerv1.VerifyEmailRequest{Token: "t"})
			return err
		},
		"ResendVerifyEmail": func(c0 context.Context) error {
			_, err := c.ResendVerifyEmail(c0, &authorizerv1.ResendVerifyEmailRequest{Email: "x@example.com", Identifier: "id"})
			return err
		},
		"VerifyOtp": func(c0 context.Context) error {
			_, err := c.VerifyOtp(c0, &authorizerv1.VerifyOtpRequest{Email: "x@example.com", Otp: "1"})
			return err
		},
		"ResendOtp": func(c0 context.Context) error {
			_, err := c.ResendOtp(c0, &authorizerv1.ResendOtpRequest{Email: "x@example.com"})
			return err
		},
		"ForgotPassword": func(c0 context.Context) error {
			_, err := c.ForgotPassword(c0, &authorizerv1.ForgotPasswordRequest{Email: "x@example.com"})
			return err
		},
		"ResetPassword": func(c0 context.Context) error {
			_, err := c.ResetPassword(c0, &authorizerv1.ResetPasswordRequest{Token: "t", Password: "p", ConfirmPassword: "p"})
			return err
		},
		"UpdateProfile": func(c0 context.Context) error {
			_, err := c.UpdateProfile(c0, &authorizerv1.UpdateProfileRequest{GivenName: "x"})
			return err
		},
		"DeactivateAccount": func(c0 context.Context) error {
			_, err := c.DeactivateAccount(c0, &authorizerv1.DeactivateAccountRequest{})
			return err
		},
	}

	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			err := fn(ctx)
			if err == nil {
				// A deliberately-generic success (enumeration-safe path) is
				// itself proof the handler is wired.
				return
			}
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.NotEqual(t, codes.Unimplemented, st.Code(),
				"AuthorizerService.%s is migrated and must not return Unimplemented", name)
		})
	}
}

// TestAuthorizerServicePermissionRPCsAreImplemented is the positive counterpart
// to the stub contract: CheckPermissions and ListPermissions replaced the old
// Permissions RPC and are wired to the service layer. They MUST NOT report
// Unimplemented. With no FGA engine configured (the default test setup) they
// fail closed — but with a real status code, never codes.Unimplemented.
func TestAuthorizerServicePermissionRPCsAreImplemented(t *testing.T) {
	conn := bootGRPCBufconn(t)
	ctx := context.Background()
	c := authorizerv1.NewAuthorizerServiceClient(conn)

	type call func(context.Context) error
	cases := map[string]call{
		"CheckPermissions": func(c0 context.Context) error {
			_, err := c.CheckPermissions(c0, &authorizerv1.CheckPermissionsRequest{
				Checks: []*authorizerv1.PermissionCheckInput{{Relation: "can_view", Object: "document:1"}},
			})
			return err
		},
		"ListPermissions": func(c0 context.Context) error {
			_, err := c.ListPermissions(c0, &authorizerv1.ListPermissionsRequest{})
			return err
		},
	}

	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			err := fn(ctx)
			require.Error(t, err, "permission RPC should fail closed without an FGA engine")
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.NotEqual(t, codes.Unimplemented, st.Code(),
				"AuthorizerService.%s is implemented and must not return Unimplemented", name)
		})
	}
}

// TestGRPCHealthCheckProtocol exercises the standard grpc.health.v1.Health
// service that the gRPC server registers for k8s readiness probes.
func TestGRPCHealthCheckProtocol(t *testing.T) {
	conn := bootGRPCBufconn(t)
	resp, err := healthv1.NewHealthClient(conn).Check(context.Background(), &healthv1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, healthv1.HealthCheckResponse_SERVING, resp.Status)
}

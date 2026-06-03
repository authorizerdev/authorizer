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

// TestAuthorizerServiceStubsReturnUnimplemented locks down the contract for
// every not-yet-migrated method on the consolidated AuthorizerService.
// Real today: Meta, Profile, Permissions, Logout, Revoke, Session,
// ValidateJwtToken, ValidateSession (covered elsewhere). As each remaining
// method's handler is wired up, drop its entry below.
func TestAuthorizerServiceStubsReturnUnimplemented(t *testing.T) {
	conn := bootGRPCBufconn(t)
	ctx := context.Background()
	c := authorizerv1.NewAuthorizerServiceClient(conn)

	type call func(context.Context) error
	cases := map[string]call{
		"Signup": func(c0 context.Context) error {
			_, err := c.Signup(c0, &authorizerv1.SignupRequest{Password: "p", ConfirmPassword: "p"})
			return err
		},
		"Login": func(c0 context.Context) error {
			_, err := c.Login(c0, &authorizerv1.LoginRequest{Password: "p"})
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
			_, err := c.UpdateProfile(c0, &authorizerv1.UpdateProfileRequest{})
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
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.Unimplemented, st.Code(),
				"stub for AuthorizerService.%s should return Unimplemented until its handler is wired", name)
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

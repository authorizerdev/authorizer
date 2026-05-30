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

	authzv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/authz/v1"
	sessionv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/session/v1"
	tokenv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/token/v1"
	userv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/user/v1"
	verificationv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/verification/v1"
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

// TestGRPCStubsReturnUnimplemented locks down the Phase 2 contract: every
// service is registered (so reflection sees it) but the non-migrated ones
// return codes.Unimplemented until their handlers replace the stubs in
// follow-up PRs. A regression — e.g. accidentally returning OK or panicking
// — would silently change client behaviour.
func TestGRPCStubsReturnUnimplemented(t *testing.T) {
	conn := bootGRPCBufconn(t)
	ctx := context.Background()

	type call func(context.Context) error
	cases := map[string]call{
		"UserService.CreateUser": func(c context.Context) error {
			_, err := userv1.NewUserServiceClient(conn).CreateUser(c, &userv1.CreateUserRequest{
				Email: "x@example.com", Password: "p", ConfirmPassword: "p",
			})
			return err
		},
		"UserService.GetUser": func(c context.Context) error {
			_, err := userv1.NewUserServiceClient(conn).GetUser(c, &userv1.GetUserRequest{Name: "users/me"})
			return err
		},
		"UserService.UpdateUser": func(c context.Context) error {
			_, err := userv1.NewUserServiceClient(conn).UpdateUser(c, &userv1.UpdateUserRequest{User: &userv1.User{Id: "users/me"}})
			return err
		},
		"UserService.DeleteUser": func(c context.Context) error {
			_, err := userv1.NewUserServiceClient(conn).DeleteUser(c, &userv1.DeleteUserRequest{Name: "users/me"})
			return err
		},
		"SessionService.CreateSession": func(c context.Context) error {
			_, err := sessionv1.NewSessionServiceClient(conn).CreateSession(c, &sessionv1.CreateSessionRequest{
				Grant: &sessionv1.CreateSessionRequest_Password{
					Password: &sessionv1.PasswordGrant{Email: "x@example.com", Password: "p"},
				},
			})
			return err
		},
		"SessionService.GetCurrentSession": func(c context.Context) error {
			_, err := sessionv1.NewSessionServiceClient(conn).GetCurrentSession(c, &sessionv1.GetCurrentSessionRequest{})
			return err
		},
		"SessionService.DeleteSession": func(c context.Context) error {
			_, err := sessionv1.NewSessionServiceClient(conn).DeleteSession(c, &sessionv1.DeleteSessionRequest{})
			return err
		},
		"SessionService.CreateSessionValidation": func(c context.Context) error {
			_, err := sessionv1.NewSessionServiceClient(conn).CreateSessionValidation(c, &sessionv1.CreateSessionValidationRequest{Cookie: "x"})
			return err
		},
		"MagicLinkService.CreateMagicLink": func(c context.Context) error {
			_, err := sessionv1.NewMagicLinkServiceClient(conn).CreateMagicLink(c, &sessionv1.CreateMagicLinkRequest{Email: "x@example.com"})
			return err
		},
		"EmailVerification.Create": func(c context.Context) error {
			_, err := verificationv1.NewEmailVerificationServiceClient(conn).CreateEmailVerification(c, &verificationv1.CreateEmailVerificationRequest{
				Email: "x@example.com", Identifier: "id",
			})
			return err
		},
		"EmailVerification.Confirm": func(c context.Context) error {
			_, err := verificationv1.NewEmailVerificationServiceClient(conn).ConfirmEmailVerification(c, &verificationv1.ConfirmEmailVerificationRequest{Token: "t"})
			return err
		},
		"PasswordReset.Create": func(c context.Context) error {
			_, err := verificationv1.NewPasswordResetServiceClient(conn).CreatePasswordReset(c, &verificationv1.CreatePasswordResetRequest{Email: "x@example.com"})
			return err
		},
		"PasswordReset.Confirm": func(c context.Context) error {
			_, err := verificationv1.NewPasswordResetServiceClient(conn).ConfirmPasswordReset(c, &verificationv1.ConfirmPasswordResetRequest{Token: "t", Password: "p", ConfirmPassword: "p"})
			return err
		},
		"OtpChallenge.Create": func(c context.Context) error {
			_, err := verificationv1.NewOtpChallengeServiceClient(conn).CreateOtpChallenge(c, &verificationv1.CreateOtpChallengeRequest{Email: "x@example.com"})
			return err
		},
		"OtpChallenge.Confirm": func(c context.Context) error {
			_, err := verificationv1.NewOtpChallengeServiceClient(conn).ConfirmOtpChallenge(c, &verificationv1.ConfirmOtpChallengeRequest{ChallengeId: "id", Otp: "1"})
			return err
		},
		"TokenService.CreateTokenValidation": func(c context.Context) error {
			_, err := tokenv1.NewTokenServiceClient(conn).CreateTokenValidation(c, &tokenv1.CreateTokenValidationRequest{TokenType: "access_token", Token: "t"})
			return err
		},
		"TokenService.RevokeRefreshToken": func(c context.Context) error {
			_, err := tokenv1.NewTokenServiceClient(conn).RevokeRefreshToken(c, &tokenv1.RevokeRefreshTokenRequest{RefreshToken: "t"})
			return err
		},
		"AuthzService.ListMyPermissions": func(c context.Context) error {
			_, err := authzv1.NewAuthzServiceClient(conn).ListMyPermissions(c, &authzv1.ListMyPermissionsRequest{})
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
				"stub for %s should return Unimplemented until its handler is wired", name)
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

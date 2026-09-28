package integration_tests

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
)

// publicClientForSetup is newPublicClient's counterpart for a caller that
// already owns a testSetup — needed here because the delegated token must be
// minted through the SAME setup's /oauth/token that the gRPC server serves.
func publicClientForSetup(t *testing.T, ts *testSetup) authorizerv1.AuthorizerServiceClient {
	t.Helper()

	srv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             ts.Logger,
		Config:          ts.Config,
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

	return authorizerv1.NewAuthorizerServiceClient(conn)
}

// TestPublicRPCsEnforceDelegatedScope pins that the `public` proto annotation
// means authentication is OPTIONAL — never that authorization is skipped.
//
// The gRPC auth interceptor used to return `handler(ctx, req)` for every public
// method before it resolved any credential, so enforceDelegatedScope never ran
// on that path (GHSA-25j3-4jg3-w352). Six public RPCs also accept a bearer
// token and treat it as the user's own identity, so an agent holding an
// `openid`-scoped delegated token could call TotpMfaSetup, receive a Verified
// MFA-session cookie, and trade it at SkipMfaSetup for an unattenuated
// first-party user token with no `act` claim.
//
// GraphQL was never affected: its gate is a resolver middleware with no notion
// of a public operation. This test exists because the gRPC/REST side had no
// equivalent coverage, which is how the gap survived.
func TestPublicRPCsEnforceDelegatedScope(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	tokenRouter := gin.New()
	tokenRouter.POST("/oauth/token", ts.HttpProvider.TokenHandler())
	delegated, _, _ := mintDelegatedViaEndpoint(t, ts, tokenRouter, testAuthorizerHost(ts))

	c := publicClientForSetup(t, ts)

	// The delegated token is minted with scope openid/profile/email, and no MFA
	// RPC is on the delegated allow-list at all, so every one of these must fail
	// closed regardless of scope.
	delegatedCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer "+delegated,
		"x-authorizer-url", testAuthorizerHost(ts),
	))

	t.Run("TotpMfaSetup", func(t *testing.T) {
		_, err := c.TotpMfaSetup(delegatedCtx, &authorizerv1.TotpMfaSetupRequest{})
		require.Error(t, err)
		require.Equal(t, codes.PermissionDenied, status.Code(err),
			"a delegated token must not reach a public bearer-capable MFA setup RPC")
	})

	t.Run("EmailOtpMfaSetup", func(t *testing.T) {
		_, err := c.EmailOtpMfaSetup(delegatedCtx, &authorizerv1.EmailOtpMfaSetupRequest{})
		require.Error(t, err)
		require.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("WebauthnRegistrationOptions", func(t *testing.T) {
		_, err := c.WebauthnRegistrationOptions(delegatedCtx, &authorizerv1.WebauthnRegistrationOptionsRequest{})
		require.Error(t, err)
		require.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("SkipMfaSetup", func(t *testing.T) {
		_, err := c.SkipMfaSetup(delegatedCtx, &authorizerv1.SkipMfaSetupRequest{})
		require.Error(t, err)
		require.Equal(t, codes.PermissionDenied, status.Code(err),
			"the second half of the chain must be closed too, not just the first")
	})

	// The reported surface was REST /v1/*, not native gRPC. The gateway
	// dispatches through the same interceptor, so this passes for the same
	// reason — but the regression test should pin the surface the report
	// actually used, not only the one that was convenient to drive.
	t.Run("REST /v1/totp_mfa_setup", func(t *testing.T) {
		baseURL := newAdminRESTServer(t, ts)
		req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/totp_mfa_setup", strings.NewReader("{}"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+delegated)
		req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusForbidden, resp.StatusCode,
			"the REST gateway must refuse a delegated caller here too")
		for _, c := range resp.Cookies() {
			require.NotContains(t, c.Name, "mfa", "no MFA session may be handed out")
		}
	})
}

// TestMachineTokenCannotEnrollMFA covers the case enforceDelegatedScope cannot:
// a pure client_credentials token carries no `act` claim, so ImmediateActor
// returns "" and the delegated gate deliberately ignores it.
//
// It was nonetheless accepted as a credential by the same dual-mode resolvers,
// and the chain only died further in, on GetUserByID missing — a service
// account is a Client row, not a User row. That produced Internal/500 for what
// is really a rejected credential, and it held only for as long as no service
// account has a user row. The resolvers now reject the machine login_method
// outright.
func TestMachineTokenCannotEnrollMFA(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	tokenRouter := gin.New()
	tokenRouter.POST("/oauth/token", ts.HttpProvider.TokenHandler())
	clientID, secret := newDelegationAgent(t, ts, "openid,profile,email")
	machine := agentAccessToken(t, ts, tokenRouter, clientID, secret)

	c := publicClientForSetup(t, ts)
	machineCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer "+machine,
		"x-authorizer-url", testAuthorizerHost(ts),
	))

	t.Run("TotpMfaSetup", func(t *testing.T) {
		_, err := c.TotpMfaSetup(machineCtx, &authorizerv1.TotpMfaSetupRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err),
			"a machine identity must be rejected as unauthorized, not surface as Internal")
	})

	t.Run("WebauthnRegistrationOptions", func(t *testing.T) {
		_, err := c.WebauthnRegistrationOptions(machineCtx, &authorizerv1.WebauthnRegistrationOptionsRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

// TestPublicRPCsRemainReachableWithoutCredentials is the other half of the
// contract, and the regression the fix above is most likely to cause: a public
// RPC must still work for a caller presenting NO credential, and must not turn
// into Unauthenticated merely because a stale cookie or expired bearer happened
// to be attached.
func TestPublicRPCsRemainReachableWithoutCredentials(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	c := publicClientForSetup(t, ts)

	email := "public_rpc_nocred_" + uuid.New().String() + "@authorizer.dev"
	const password = "Password@123"

	t.Run("no credential at all", func(t *testing.T) {
		_, err := c.Signup(context.Background(), &authorizerv1.SignupRequest{
			Email:           email,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err, "a public RPC must be reachable unauthenticated")
	})

	t.Run("a garbage bearer must not make a public RPC Unauthenticated", func(t *testing.T) {
		staleCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
			"authorization", "Bearer not-a-real-token",
			"x-authorizer-url", testAuthorizerHost(ts),
		))
		_, err := c.Login(staleCtx, &authorizerv1.LoginRequest{Email: email, Password: password})
		// Login may succeed or fail on credentials, but it must not be rejected
		// by the interceptor for carrying an unusable token.
		if err != nil {
			require.NotEqual(t, codes.Unauthenticated, status.Code(err),
				"an unusable credential must be ignored on a public RPC, not rejected")
			require.NotEqual(t, codes.PermissionDenied, status.Code(err))
		}
	})
}

// TestFirstPartyTokenStillReachesMFASetup guards the legitimate path: an
// ordinary logged-in user adding a second factor from account settings carries
// a first-party bearer, which enforceDelegatedScope deliberately does not gate.
func TestFirstPartyTokenStillReachesMFASetup(t *testing.T) {
	// MFA deliberately left off: enabling it makes login withhold the token
	// behind the first-time-offer gate, and the assertion here is about the
	// INTERCEPTOR, not about TOTP being available. A first-party caller may
	// legitimately get FailedPrecondition from the service; what it must never
	// get is PermissionDenied from the delegated-scope gate.
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	c := publicClientForSetup(t, ts)

	email := "public_rpc_firstparty_" + uuid.New().String() + "@authorizer.dev"
	const password = "Password@123"

	_, err := c.Signup(context.Background(), &authorizerv1.SignupRequest{
		Email:           email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)

	loginRes, err := c.Login(context.Background(), &authorizerv1.LoginRequest{
		Email:    email,
		Password: password,
	})
	require.NoError(t, err)
	require.NotEmpty(t, loginRes.GetAccessToken())

	firstPartyCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"authorization", "Bearer "+loginRes.GetAccessToken(),
		"x-authorizer-url", testAuthorizerHost(ts),
	))

	_, err = c.TotpMfaSetup(firstPartyCtx, &authorizerv1.TotpMfaSetupRequest{})
	if err != nil {
		require.NotEqual(t, codes.PermissionDenied, status.Code(err),
			"a first-party token must not be scope-gated on the MFA setup path")
	}
}

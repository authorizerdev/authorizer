package integration_tests

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/gateway"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
)

// bootRESTGateway boots the gRPC server + grpc-gateway over an httptest server,
// mirroring the production wiring (the gateway mounted under the gin router at
// /v1/*). It uses the fully-wired service from initTestSetup so auth-bearing
// methods (Profile, Logout, etc.) have real Token/Storage providers rather
// than nil stubs. Returns the base URL.
func bootRESTGateway(t *testing.T) string {
	t.Helper()
	cfg := getTestConfig()
	cfg.ClientID = "test-client"

	s := initTestSetup(t, cfg)

	grpcSrv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             s.Logger,
		Config:          cfg,
		ServiceProvider: s.ServiceProvider,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	gw, cleanup, err := gateway.Handler(ctx, grpcSrv.GRPCServer())
	require.NoError(t, err)
	t.Cleanup(cleanup)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/v1/*path", gin.WrapH(gw))
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts.URL
}

// decodeErrorEnvelope reads the standard REST error envelope.
type errorEnvelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func readEnvelope(t *testing.T, resp *http.Response) errorEnvelope {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var env errorEnvelope
	require.NoError(t, json.Unmarshal(body, &env), "body: %s", string(body))
	return env
}

// TestRESTErrorEnvelopeAndStatusCodes verifies that business errors surface as
// proper HTTP status codes (not a blanket 500) and in the stable snake_case
// error envelope {"code": ..., "message": ...}. Exercises the service typed
// errors -> ErrorMap interceptor -> grpc-gateway error handler chain.
func TestRESTErrorEnvelopeAndStatusCodes(t *testing.T) {
	base := bootRESTGateway(t)

	t.Run("signup missing email and phone -> 400 invalid_argument", func(t *testing.T) {
		resp, err := http.Post(base+"/v1/signup", "application/json",
			strings.NewReader(`{"password":"Test@123","confirm_password":"Test@123"}`))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		env := readEnvelope(t, resp)
		require.Equal(t, "invalid_argument", env.Code)
		require.Equal(t, "email or phone number is required", env.Message)
	})

	t.Run("profile without auth -> 401 unauthenticated", func(t *testing.T) {
		resp, err := http.Get(base + "/v1/profile")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		env := readEnvelope(t, resp)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("validate_jwt_token bad token type -> 400 invalid_argument", func(t *testing.T) {
		resp, err := http.Post(base+"/v1/validate_jwt_token", "application/json",
			strings.NewReader(`{"token_type":"bogus","token":"x"}`))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		env := readEnvelope(t, resp)
		require.Equal(t, "invalid_argument", env.Code)
		require.Equal(t, "invalid token type", env.Message)
	})
}

// TestErrorMessageConsistencyAcrossProtocols asserts that the SAME business
// error surfaces with the IDENTICAL message string over GraphQL, gRPC, and
// REST. All three transports delegate to the same internal/service method and
// surface its service.Error.Error() text verbatim — GraphQL as the resolver
// error, gRPC as the status message, REST as the envelope `message`. This test
// is the regression guard for that contract.
func TestErrorMessageConsistencyAcrossProtocols(t *testing.T) {
	cfg := getTestConfig()
	cfg.ClientID = "test-client"
	s := initTestSetup(t, cfg)

	grpcSrv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             s.Logger,
		Config:          cfg,
		ServiceProvider: s.ServiceProvider,
	})
	require.NoError(t, err)

	// gRPC client over an in-process bufconn served by the same server.
	lis := bufconn.Listen(1 << 20)
	t.Cleanup(func() { _ = lis.Close() })
	go func() { _ = grpcSrv.GRPCServer().Serve(lis) }()
	t.Cleanup(grpcSrv.GRPCServer().GracefulStop)
	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	// REST gateway over the same gRPC server.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	gw, cleanup, err := gateway.Handler(ctx, grpcSrv.GRPCServer())
	require.NoError(t, err)
	t.Cleanup(cleanup)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/v1/*path", gin.WrapH(gw))
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	// "email or phone number is required" — the first validation in SignUp,
	// reached identically regardless of transport.
	const wantMsg = "email or phone number is required"

	// GraphQL.
	_, ctxGin := createContext(s)
	_, gqlErr := s.GraphQLProvider.SignUp(ctxGin, &model.SignUpRequest{
		Password:        "Test@123",
		ConfirmPassword: "Test@123",
	})
	require.Error(t, gqlErr)
	require.Equal(t, wantMsg, gqlErr.Error(), "GraphQL error message")

	// gRPC.
	_, grpcErr := authorizerv1.NewAuthorizerServiceClient(conn).Signup(context.Background(),
		&authorizerv1.SignupRequest{Password: "Test@123", ConfirmPassword: "Test@123"})
	require.Error(t, grpcErr)
	require.Equal(t, wantMsg, status.Convert(grpcErr).Message(), "gRPC status message")

	// REST.
	resp, err := http.Post(ts.URL+"/v1/signup", "application/json",
		strings.NewReader(`{"password":"Test@123","confirm_password":"Test@123"}`))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, wantMsg, readEnvelope(t, resp).Message, "REST envelope message")
}

// TestRESTLogoutIsPost asserts logout is mapped to POST (a mutating, audited
// operation must not be a safe GET). GET must not be routed to the handler.
func TestRESTLogoutIsPost(t *testing.T) {
	base := bootRESTGateway(t)

	// GET is not a registered method for /v1/logout -> gateway returns 405.
	getResp, err := http.Get(base + "/v1/logout")
	require.NoError(t, err)
	defer func() { _ = getResp.Body.Close() }()
	require.Equal(t, http.StatusMethodNotAllowed, getResp.StatusCode)

	// POST reaches the handler; with no session it is unauthenticated (401),
	// proving the route exists and errors flow through the envelope.
	postResp, err := http.Post(base+"/v1/logout", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer func() { _ = postResp.Body.Close() }()
	require.Equal(t, http.StatusUnauthorized, postResp.StatusCode)
	env := readEnvelope(t, postResp)
	require.Equal(t, "unauthenticated", env.Code)
}

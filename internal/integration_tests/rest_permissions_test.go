package integration_tests

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/service"
)

// TestRESTCheckPermissionsFailClosed mirrors the gRPC fail-closed contract over
// the grpc-gateway REST surface: POST /v1/check_permissions with no FGA engine
// configured returns the service.ErrFgaNotEnabled error. That plain error maps
// to codes.FailedPrecondition -> HTTP 400, rendered in the stable snake_case
// envelope {"code": "failed_precondition", "message": "..."}.
//
// The engine-nil guard is the first check in the service method, so even this
// unauthenticated request surfaces the FGA-disabled error rather than a 401.
func TestRESTCheckPermissionsFailClosed(t *testing.T) {
	base := bootRESTGateway(t)

	resp, err := http.Post(base+"/v1/check_permissions", "application/json",
		strings.NewReader(`{"checks":[{"relation":"can_view","object":"document:1"}]}`))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	env := readEnvelope(t, resp)
	require.Equal(t, "failed_precondition", env.Code)
	require.Equal(t, service.ErrFgaNotEnabled.Error(), env.Message)
}

// TestRESTListPermissionsFailClosed is the ListPermissions counterpart: with no
// FGA engine, POST /v1/list_permissions fails closed with the same envelope.
func TestRESTListPermissionsFailClosed(t *testing.T) {
	base := bootRESTGateway(t)

	resp, err := http.Post(base+"/v1/list_permissions", "application/json",
		strings.NewReader(`{}`))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	env := readEnvelope(t, resp)
	require.Equal(t, "failed_precondition", env.Code)
	require.Equal(t, service.ErrFgaNotEnabled.Error(), env.Message)
}

// TestRESTCheckPermissionsEmptyChecksRejected verifies the protovalidate
// min_items=1 constraint on CheckPermissionsRequest.checks is enforced over
// REST too: an empty checks array is rejected by the validate interceptor with
// codes.InvalidArgument -> HTTP 400, before the handler/engine guard runs.
func TestRESTCheckPermissionsEmptyChecksRejected(t *testing.T) {
	base := bootRESTGateway(t)

	resp, err := http.Post(base+"/v1/check_permissions", "application/json",
		strings.NewReader(`{"checks":[]}`))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	env := readEnvelope(t, resp)
	require.Equal(t, "invalid_argument", env.Code)
}

// TestRESTPermissionsAreNotGet asserts the permission RPCs are mapped to POST
// (they evaluate FGA decisions and accept a request body). A GET on the path
// must not route to the handler — the gateway returns 405.
func TestRESTPermissionsAreNotGet(t *testing.T) {
	base := bootRESTGateway(t)

	for _, path := range []string{"/v1/check_permissions", "/v1/list_permissions"} {
		resp, err := http.Get(base + path)
		require.NoError(t, err)
		require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "GET %s should be 405", path)
		_ = resp.Body.Close()
	}
}

// TestRESTGatewayForwardsAuthorizerHost is the regression test for the
// gateway's WithMetadata annotator. The in-process bufconn call carries
// ":authority=bufconn"; without forwarding the original request's resolved
// host as `x-authorizer-url` metadata, tokens minted over REST would carry
// iss=http://bufconn and JWT issuer validation would reject every token on
// any surface. The test signs up over REST with an explicit X-Authorizer-URL,
// asserts the minted access token's iss claim echoes it, then proves the
// token round-trips: GET /v1/profile with the same host header returns the
// authenticated user.
func TestRESTGatewayForwardsAuthorizerHost(t *testing.T) {
	base := bootRESTGateway(t)
	const hostURL = "http://auth.test.example"

	body := strings.NewReader(`{"email":"rest-host@test.dev","password":"Rest-Host-123!","confirm_password":"Rest-Host-123!"}`)
	req, err := http.NewRequest(http.MethodPost, base+"/v1/signup", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Authorizer-URL", hostURL)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var signup struct {
		AccessToken string `json:"access_token"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&signup))
	require.NotEmpty(t, signup.AccessToken)

	// The iss claim must be the host the gateway forwarded — not the
	// in-process bufconn authority.
	parts := strings.Split(signup.AccessToken, ".")
	require.Len(t, parts, 3)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims struct {
		Iss string `json:"iss"`
	}
	require.NoError(t, json.Unmarshal(payload, &claims))
	require.Equal(t, hostURL, claims.Iss,
		"gateway must forward the resolved authorizer host; got iss=%q", claims.Iss)

	// Round-trip: the REST-minted token authenticates a REST identity call.
	preq, err := http.NewRequest(http.MethodGet, base+"/v1/profile", nil)
	require.NoError(t, err)
	preq.Header.Set("Authorization", "Bearer "+signup.AccessToken)
	preq.Header.Set("X-Authorizer-URL", hostURL)
	presp, err := http.DefaultClient.Do(preq)
	require.NoError(t, err)
	defer func() { _ = presp.Body.Close() }()
	require.Equal(t, http.StatusOK, presp.StatusCode)
	var profile struct {
		Email string `json:"email"`
	}
	require.NoError(t, json.NewDecoder(presp.Body).Decode(&profile))
	require.Equal(t, "rest-host@test.dev", profile.Email)
}

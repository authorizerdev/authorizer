package integration_tests

import (
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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

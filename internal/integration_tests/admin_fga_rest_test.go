package integration_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// The admin FGA REST tests run against the standard admin test harness, which
// wires NO authorization engine (getTestConfig sets no --fga-store). That gives
// every op two deterministic assertions over the REST (grpc-gateway) surface
// without standing up a real FGA store:
//
//   - AUTH fail-closed: no admin secret => HTTP 401 {"code":"unauthenticated"}.
//   - ENGINE fail-closed: with a valid admin secret but no engine configured =>
//     HTTP 400 {"code":"failed_precondition"} (codes.FailedPrecondition mapped by
//     the gateway). This is the security-critical "fail closed when FGA is not
//     configured" contract. Mirrors the gRPC FGA fail-closed tests.

// fgaRESTCase describes one FGA REST endpoint and a valid-shape request body so
// the engine-not-configured contract can be asserted without a real store.
type fgaRESTCase struct {
	name   string
	method string
	path   string
	body   string
}

// TestAdminFgaREST asserts, for each of the 8 admin FGA REST endpoints, the auth
// fail-closed contract (no secret -> 401 unauthenticated) and the
// engine-not-configured contract (valid secret, no engine -> 400
// failed_precondition).
func TestAdminFgaREST(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	baseURL := newAdminRESTServer(t, ts)
	secret := cfg.AdminSecret

	cases := []fgaRESTCase{
		{
			name:   "get model",
			method: http.MethodGet,
			path:   "/v1/admin/fga/model",
			body:   "",
		},
		{
			name:   "write model",
			method: http.MethodPost,
			path:   "/v1/admin/fga/model",
			body:   `{"dsl":"model\n  schema 1.1\ntype user"}`,
		},
		{
			name:   "write tuples",
			method: http.MethodPost,
			path:   "/v1/admin/fga/tuples",
			body:   `{"tuples":[{"user":"user:alice","relation":"owner","object":"document:1"}]}`,
		},
		{
			name:   "delete tuples",
			method: http.MethodPost,
			path:   "/v1/admin/fga/tuples/delete",
			body:   `{"tuples":[{"user":"user:alice","relation":"owner","object":"document:1"}]}`,
		},
		{
			name:   "read tuples",
			method: http.MethodPost,
			path:   "/v1/admin/fga/tuples/read",
			body:   "{}",
		},
		{
			name:   "list users",
			method: http.MethodPost,
			path:   "/v1/admin/fga/list_users",
			body:   `{"object":"document:1","relation":"viewer","user_type":"user"}`,
		},
		{
			name:   "expand",
			method: http.MethodPost,
			path:   "/v1/admin/fga/expand",
			body:   `{"relation":"viewer","object":"document:1"}`,
		},
		{
			name:   "reset",
			method: http.MethodPost,
			path:   "/v1/admin/fga/reset",
			body:   "{}",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Run("fail closed without admin secret", func(t *testing.T) {
				var env struct {
					Code string `json:"code"`
				}
				status := adminRESTJSON(t, baseURL, tc.method, tc.path, "", tc.body, &env)
				require.Equal(t, http.StatusUnauthorized, status)
				require.Equal(t, "unauthenticated", env.Code)
			})

			t.Run("fail closed when FGA engine not configured", func(t *testing.T) {
				var env struct {
					Code string `json:"code"`
				}
				status := adminRESTJSON(t, baseURL, tc.method, tc.path, secret, tc.body, &env)
				require.Equal(t, http.StatusBadRequest, status)
				require.Equal(t, "failed_precondition", env.Code)
			})
		})
	}
}

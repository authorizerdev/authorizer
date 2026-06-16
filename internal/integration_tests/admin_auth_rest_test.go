package integration_tests

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/stretchr/testify/require"
)

// TestAdminAuthREST exercises the admin auth + meta operations over the REST
// (grpc-gateway) surface, asserting the fail-closed contract (no admin secret ->
// 401) and the happy path (x-authorizer-admin-secret header) for each.
func TestAdminAuthREST(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	baseURL := newAdminRESTServer(t, ts)
	secret := cfg.AdminSecret

	t.Run("meta fail-closed", func(t *testing.T) {
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodGet, "/v1/admin/meta", "", "", &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("meta happy path", func(t *testing.T) {
		var out struct {
			AdminMeta struct {
				Roles []string `json:"roles"`
			} `json:"admin_meta"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodGet, "/v1/admin/meta", secret, "", &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, cfg.Roles, out.AdminMeta.Roles)
	})

	t.Run("login invalid secret", func(t *testing.T) {
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/login", "",
			`{"admin_secret":"wrong-secret"}`, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("login happy path", func(t *testing.T) {
		var out struct {
			Message string `json:"message"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/login", "",
			fmt.Sprintf(`{"admin_secret":%q}`, secret), &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "admin logged in successfully", out.Message)
	})

	t.Run("login returns a real Set-Cookie header", func(t *testing.T) {
		// Regression: admin login sets the admin session cookie as gRPC
		// `set-cookie` metadata; the gateway's outgoing-header matcher must
		// promote it to a real Set-Cookie header (not Grpc-Metadata-Set-Cookie).
		req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/admin/login", strings.NewReader(
			fmt.Sprintf(`{"admin_secret":%q}`, secret)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		cookies := resp.Header.Values("Set-Cookie")
		require.NotEmpty(t, cookies, "admin login must return a real Set-Cookie header over REST")
		found := false
		for _, c := range cookies {
			if strings.HasPrefix(c, constants.AdminCookieName+"=") {
				found = true
				break
			}
		}
		require.True(t, found, "expected %s cookie; got %v", constants.AdminCookieName, cookies)
		require.Empty(t, resp.Header.Values("Grpc-Metadata-Set-Cookie"),
			"raw gRPC metadata cookie header must not leak over REST")
	})

	t.Run("session fail-closed then happy", func(t *testing.T) {
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodGet, "/v1/admin/session", "", "", &env)
		require.Equal(t, http.StatusUnauthorized, status)

		var out struct {
			Message string `json:"message"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodGet, "/v1/admin/session", secret, "", &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "admin session refreshed successfully", out.Message)
	})

	t.Run("logout fail-closed then happy", func(t *testing.T) {
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/logout", "", "", &env)
		require.Equal(t, http.StatusUnauthorized, status)

		var out struct {
			Message string `json:"message"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/logout", secret, "", &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "admin logged out successfully", out.Message)
	})
}

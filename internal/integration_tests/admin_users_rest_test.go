package integration_tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAdminUsersREST exercises the admin user-management operations over the
// REST (grpc-gateway) surface, mirroring TestAdminUsersGRPC and friends. Each
// subtest asserts the fail-closed contract (no admin secret -> 401) and the
// happy path (x-authorizer-admin-secret header). Data is seeded via the shared
// StorageProvider before the REST calls so it is visible to the in-process
// gRPC server.
func TestAdminUsersREST(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	baseURL := newAdminRESTServer(t, ts)
	secret := cfg.AdminSecret

	t.Run("users fail-closed", func(t *testing.T) {
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/users", "", "{}", &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("users happy path", func(t *testing.T) {
		_, email := seedUser(t, ts)
		var out struct {
			Users []struct {
				Email string `json:"email"`
			} `json:"users"`
			Pagination map[string]any `json:"pagination"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/users", secret, "{}", &out)
		require.Equal(t, http.StatusOK, status)
		require.NotNil(t, out.Pagination)
		var found bool
		for _, u := range out.Users {
			if u.Email == email {
				found = true
				break
			}
		}
		require.True(t, found, "seeded user should appear in the users page")
	})

	t.Run("user fail-closed", func(t *testing.T) {
		id, _ := seedUser(t, ts)
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/user", "",
			fmt.Sprintf(`{"id":%q}`, id), &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("user happy path by id", func(t *testing.T) {
		id, email := seedUser(t, ts)
		var out struct {
			User struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			} `json:"user"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/user", secret,
			fmt.Sprintf(`{"id":%q}`, id), &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, id, out.User.ID)
		require.Equal(t, email, out.User.Email)
	})

	t.Run("update_user fail-closed", func(t *testing.T) {
		id, _ := seedUser(t, ts)
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/update_user", "",
			fmt.Sprintf(`{"id":%q,"given_name":"Ada"}`, id), &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("update_user happy path", func(t *testing.T) {
		id, _ := seedUser(t, ts)
		var out struct {
			User struct {
				GivenName string `json:"given_name"`
			} `json:"user"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/update_user", secret,
			fmt.Sprintf(`{"id":%q,"given_name":"Ada"}`, id), &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "Ada", out.User.GivenName)
	})

	t.Run("delete_user fail-closed", func(t *testing.T) {
		_, email := seedUser(t, ts)
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/delete_user", "",
			fmt.Sprintf(`{"email":%q}`, email), &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("delete_user happy path", func(t *testing.T) {
		_, email := seedUser(t, ts)
		var out struct {
			Message string `json:"message"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/delete_user", secret,
			fmt.Sprintf(`{"email":%q}`, email), &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "user deleted successfully", out.Message)
	})

	t.Run("verification_requests fail-closed", func(t *testing.T) {
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/verification_requests", "", "{}", &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("verification_requests happy path", func(t *testing.T) {
		var out struct {
			Pagination map[string]any `json:"pagination"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/verification_requests", secret, "{}", &out)
		require.Equal(t, http.StatusOK, status)
		require.NotNil(t, out.Pagination)
	})
}

package integration_tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestAdminAccessREST exercises the admin access-control operations over the
// REST (grpc-gateway) surface, mirroring TestAdminRevokeAccessGRPC,
// TestAdminEnableAccessGRPC and TestAdminInviteMembersGRPC. Each subtest asserts
// the fail-closed contract (no admin secret -> 401) and the happy path
// (x-authorizer-admin-secret header). Users are seeded via the shared
// StorageProvider before the REST calls.
func TestAdminAccessREST(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	baseURL := newAdminRESTServer(t, ts)
	secret := cfg.AdminSecret

	t.Run("revoke_access fail-closed", func(t *testing.T) {
		id, _ := seedUser(t, ts)
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/revoke_access", "",
			fmt.Sprintf(`{"user_id":%q}`, id), &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("revoke_access happy path", func(t *testing.T) {
		id, _ := seedUser(t, ts)
		var out struct {
			Message string `json:"message"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/revoke_access", secret,
			fmt.Sprintf(`{"user_id":%q}`, id), &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "user access revoked successfully", out.Message)
	})

	t.Run("enable_access fail-closed", func(t *testing.T) {
		id, _ := seedUser(t, ts)
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/enable_access", "",
			fmt.Sprintf(`{"user_id":%q}`, id), &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("enable_access happy path", func(t *testing.T) {
		id, _ := seedUser(t, ts)
		// revoke first so enabling has an effect
		var revoke struct {
			Message string `json:"message"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/revoke_access", secret,
			fmt.Sprintf(`{"user_id":%q}`, id), &revoke)
		require.Equal(t, http.StatusOK, status)

		var out struct {
			Message string `json:"message"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/enable_access", secret,
			fmt.Sprintf(`{"user_id":%q}`, id), &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "user access enabled successfully", out.Message)
	})

	t.Run("invite_members fail-closed", func(t *testing.T) {
		email := "admin-invite-rest-" + uuid.New().String() + "@authorizer.test"
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/invite_members", "",
			fmt.Sprintf(`{"emails":[%q]}`, email), &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("invite_members happy path", func(t *testing.T) {
		// Match the gRPC/GraphQL invite_members pattern: the relevant feature
		// flags must be enabled on the shared config pointer before invoking.
		cfg.IsEmailServiceEnabled = true
		cfg.EnableBasicAuthentication = true
		cfg.EnableMagicLinkLogin = true

		email := "admin-invite-rest-" + uuid.New().String() + "@authorizer.test"
		var out struct {
			Users []struct {
				Email string `json:"email"`
			} `json:"users"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/invite_members", secret,
			fmt.Sprintf(`{"emails":[%q]}`, email), &out)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, out.Users, 1)
		require.Equal(t, email, out.Users[0].Email)
	})
}

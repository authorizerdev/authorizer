package integration_tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// TestAdminAuditLogsREST exercises AuthorizerAdminService.AuditLogs over the REST
// (grpc-gateway) surface: the fail-closed contract (no admin secret -> 401) and
// the happy path (x-authorizer-admin-secret header -> 200) asserting a seeded
// entry appears when filtering by its action. Mirrors TestAdminAuditLogsGRPC.
func TestAdminAuditLogsREST(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	baseURL := newAdminRESTServer(t, ts)
	secret := cfg.AdminSecret
	auditLog := seedAuditLog(t, ts, constants.AuditAdminUserUpdatedEvent)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/audit_logs", "", "{}", &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)
	})

	t.Run("returns audit logs with admin secret", func(t *testing.T) {
		var out struct {
			// grpc-gateway serializes proto int64 fields as JSON strings, so
			// pagination.total is decoded as a string here.
			Pagination *struct {
				Total string `json:"total"`
			} `json:"pagination"`
			AuditLogs []struct {
				ID     string `json:"id"`
				Action string `json:"action"`
			} `json:"audit_logs"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/audit_logs", secret, "{}", &out)
		require.Equal(t, http.StatusOK, status)
		require.NotNil(t, out.Pagination)
	})

	t.Run("filters audit logs by action with admin secret", func(t *testing.T) {
		var out struct {
			Pagination *struct {
				Total string `json:"total"`
			} `json:"pagination"`
			AuditLogs []struct {
				ID     string `json:"id"`
				Action string `json:"action"`
			} `json:"audit_logs"`
		}
		body := `{"action":"` + constants.AuditAdminUserUpdatedEvent + `"}`
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/audit_logs", secret, body, &out)
		require.Equal(t, http.StatusOK, status)
		require.NotEmpty(t, out.AuditLogs)
		found := false
		for _, l := range out.AuditLogs {
			if l.ID == auditLog.ID {
				found = true
				require.Equal(t, constants.AuditAdminUserUpdatedEvent, l.Action)
				break
			}
		}
		require.True(t, found, "seeded audit log should be present when filtering by its action")
	})
}

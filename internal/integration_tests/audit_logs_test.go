package integration_tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestAdminAuditLogs exercises the AuditLogs GraphQL resolver: it fails without
// admin auth (fail-closed) and, after admin login, returns a seeded audit log
// when filtering by its action. This is the GraphQL counterpart of the gRPC and
// REST audit tests.
func TestAdminAuditLogs(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)
	auditLog := seedAuditLog(t, ts, constants.AuditAdminUserUpdatedEvent)

	t.Run("should fail without admin cookie", func(t *testing.T) {
		res, err := ts.GraphQLProvider.AuditLogs(ctx, &model.ListAuditLogRequest{})
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should return audit logs with valid admin cookie", func(t *testing.T) {
		// Admin login first.
		_, err := ts.GraphQLProvider.AdminLogin(ctx, &model.AdminLoginRequest{
			AdminSecret: cfg.AdminSecret,
		})
		require.NoError(t, err)

		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		res, err := ts.GraphQLProvider.AuditLogs(ctx, &model.ListAuditLogRequest{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.NotNil(t, res.Pagination)
	})

	t.Run("should filter audit logs by action with valid admin cookie", func(t *testing.T) {
		action := constants.AuditAdminUserUpdatedEvent
		res, err := ts.GraphQLProvider.AuditLogs(ctx, &model.ListAuditLogRequest{
			Action: &action,
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NotEmpty(t, res.AuditLogs)
		found := false
		for _, l := range res.AuditLogs {
			if l.ID == auditLog.ID {
				found = true
				require.NotNil(t, l.Action)
				assert.Equal(t, constants.AuditAdminUserUpdatedEvent, *l.Action)
				break
			}
		}
		assert.True(t, found, "seeded audit log should be present when filtering by its action")
	})
}

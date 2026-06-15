package integration_tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// seedAuditLog inserts a deterministic audit log entry directly via storage and
// returns it. Used by the admin audit RPC test to assert a known entry appears
// in the listing.
func seedAuditLog(t *testing.T, ts *testSetup, action string) *schemas.AuditLog {
	t.Helper()
	auditLog := &schemas.AuditLog{
		ID:           uuid.New().String(),
		Action:       action,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeUser,
		ResourceID:   uuid.New().String(),
		IPAddress:    "127.0.0.1",
		UserAgent:    "test-agent",
	}
	err := ts.StorageProvider.AddAuditLog(context.Background(), auditLog)
	require.NoError(t, err)
	return auditLog
}

// TestAdminAuditLogsGRPC exercises AuthorizerAdminService.AuditLogs over gRPC:
// the fail-closed contract (no secret → Unauthenticated) and the happy path
// with a seeded entry asserted present when filtering by its action.
func TestAdminAuditLogsGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	auditLog := seedAuditLog(t, ts, constants.AuditAdminUserUpdatedEvent)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.AuditLogs(context.Background(), &authorizerv1.AuditLogsRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns audit logs with admin secret", func(t *testing.T) {
		resp, err := client.AuditLogs(adminCtx(cfg.AdminSecret), &authorizerv1.AuditLogsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp.Pagination)
	})

	t.Run("filters audit logs by action with admin secret", func(t *testing.T) {
		action := constants.AuditAdminUserUpdatedEvent
		resp, err := client.AuditLogs(adminCtx(cfg.AdminSecret), &authorizerv1.AuditLogsRequest{
			Action: &action,
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.AuditLogs)
		found := false
		for _, l := range resp.AuditLogs {
			if l.Id == auditLog.ID {
				found = true
				require.Equal(t, constants.AuditAdminUserUpdatedEvent, l.Action)
				break
			}
		}
		require.True(t, found, "seeded audit log should be present when filtering by its action")
	})
}

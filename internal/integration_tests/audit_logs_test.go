package integration_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

func TestAuditLogs(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	t.Run("should add and list audit logs", func(t *testing.T) {
		auditLog := &schemas.AuditLog{
			ActorID:        uuid.New().String(),
			ActorType:      "user",
			ActorEmail:     "test@example.com",
			Action:         "login",
			ResourceType:   "session",
			ResourceID:     uuid.New().String(),
			IPAddress:      "127.0.0.1",
			UserAgent:      "test-agent",
			OrganizationID: uuid.New().String(),
		}

		err := ts.StorageProvider.AddAuditLog(ctx, auditLog)
		require.NoError(t, err)
		assert.NotEmpty(t, auditLog.ID)
		assert.NotZero(t, auditLog.Timestamp)
		assert.NotZero(t, auditLog.CreatedAt)

		// List all audit logs
		pagination := &model.Pagination{
			Limit:  10,
			Offset: 0,
		}
		logs, pag, err := ts.StorageProvider.ListAuditLogs(ctx, pagination, map[string]interface{}{})
		require.NoError(t, err)
		assert.NotNil(t, pag)
		assert.GreaterOrEqual(t, len(logs), 1)
	})

	t.Run("should filter audit logs by action", func(t *testing.T) {
		uniqueAction := "test_action_" + uuid.New().String()[:8]

		auditLog := &schemas.AuditLog{
			ActorID:    uuid.New().String(),
			ActorType:  "user",
			ActorEmail: "filter@example.com",
			Action:     uniqueAction,
		}
		err := ts.StorageProvider.AddAuditLog(ctx, auditLog)
		require.NoError(t, err)

		pagination := &model.Pagination{
			Limit:  10,
			Offset: 0,
		}
		logs, _, err := ts.StorageProvider.ListAuditLogs(ctx, pagination, map[string]interface{}{
			"action": uniqueAction,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(logs))
		assert.Equal(t, uniqueAction, logs[0].Action)
	})

	t.Run("should filter audit logs by actor_id", func(t *testing.T) {
		actorID := uuid.New().String()

		auditLog := &schemas.AuditLog{
			ActorID:    actorID,
			ActorType:  "admin",
			ActorEmail: "admin@example.com",
			Action:     "update_env",
		}
		err := ts.StorageProvider.AddAuditLog(ctx, auditLog)
		require.NoError(t, err)

		pagination := &model.Pagination{
			Limit:  10,
			Offset: 0,
		}
		logs, _, err := ts.StorageProvider.ListAuditLogs(ctx, pagination, map[string]interface{}{
			"actor_id": actorID,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(logs))
		assert.Equal(t, actorID, logs[0].ActorID)
	})

	t.Run("should not mutate caller pagination", func(t *testing.T) {
		pagination := &model.Pagination{
			Limit:  10,
			Offset: 0,
		}
		_, returnedPag, err := ts.StorageProvider.ListAuditLogs(ctx, pagination, map[string]interface{}{})
		require.NoError(t, err)
		assert.NotSame(t, pagination, returnedPag, "should return a new pagination object")
	})

	t.Run("should delete audit logs before timestamp", func(t *testing.T) {
		uniqueAction := "cleanup_test_" + uuid.New().String()[:8]

		// Add a log with old timestamp
		oldLog := &schemas.AuditLog{
			ActorID:    uuid.New().String(),
			ActorType:  "system",
			ActorEmail: "system@example.com",
			Action:     uniqueAction,
			Timestamp:  time.Now().Add(-24 * time.Hour).Unix(),
		}
		err := ts.StorageProvider.AddAuditLog(ctx, oldLog)
		require.NoError(t, err)

		// Delete logs older than 1 hour ago
		before := time.Now().Add(-1 * time.Hour).Unix()
		err = ts.StorageProvider.DeleteAuditLogsBefore(ctx, before)
		require.NoError(t, err)

		// Verify the old log is deleted by filtering for its unique action
		pagination := &model.Pagination{
			Limit:  10,
			Offset: 0,
		}
		logs, _, err := ts.StorageProvider.ListAuditLogs(ctx, pagination, map[string]interface{}{
			"action": uniqueAction,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, len(logs))
	})
}

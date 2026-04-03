package integration_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

func TestAuditLogs(t *testing.T) {
	runForEachDB(t, func(t *testing.T, cfg *config.Config) {
		ts := initTestSetup(t, cfg)
		_, ctx := createContext(ts)

		t.Run("should add and list audit logs", func(t *testing.T) {
			auditLog := &schemas.AuditLog{
				ActorID:      uuid.New().String(),
				ActorType:    constants.AuditActorTypeUser,
				ActorEmail:   "test@example.com",
				Action:       constants.AuditLoginSuccessEvent,
				ResourceType: constants.AuditResourceTypeSession,
				ResourceID:   uuid.New().String(),
				IPAddress:    "127.0.0.1",
				UserAgent:    "test-agent",
			}

			err := ts.StorageProvider.AddAuditLog(ctx, auditLog)
			require.NoError(t, err)
			assert.NotEmpty(t, auditLog.ID)
			assert.NotZero(t, auditLog.CreatedAt)

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
				ActorID:      uuid.New().String(),
				ActorType:    constants.AuditActorTypeUser,
				ActorEmail:   "filter@example.com",
				Action:       uniqueAction,
				ResourceType: constants.AuditResourceTypeUser,
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
				ActorID:      actorID,
				ActorType:    constants.AuditActorTypeAdmin,
				ActorEmail:   "admin@example.com",
				Action:       constants.AuditAdminUserUpdatedEvent,
				ResourceType: constants.AuditResourceTypeUser,
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

		t.Run("should filter audit logs by resource_type", func(t *testing.T) {
			uniqueAction := "res_filter_" + uuid.New().String()[:8]

			auditLog := &schemas.AuditLog{
				ActorID:      uuid.New().String(),
				ActorType:    constants.AuditActorTypeAdmin,
				Action:       uniqueAction,
				ResourceType: constants.AuditResourceTypeWebhook,
				ResourceID:   uuid.New().String(),
			}
			err := ts.StorageProvider.AddAuditLog(ctx, auditLog)
			require.NoError(t, err)

			pagination := &model.Pagination{
				Limit:  10,
				Offset: 0,
			}
			logs, _, err := ts.StorageProvider.ListAuditLogs(ctx, pagination, map[string]interface{}{
				"resource_type": constants.AuditResourceTypeWebhook,
				"action":        uniqueAction,
			})
			require.NoError(t, err)
			assert.Equal(t, 1, len(logs))
			assert.Equal(t, constants.AuditResourceTypeWebhook, logs[0].ResourceType)
		})

		t.Run("should filter audit logs by timestamp range", func(t *testing.T) {
			uniqueAction := "ts_filter_" + uuid.New().String()[:8]
			now := time.Now().Unix()

			auditLog := &schemas.AuditLog{
				ActorID:      uuid.New().String(),
				ActorType:    constants.AuditActorTypeUser,
				Action:       uniqueAction,
				ResourceType: constants.AuditResourceTypeSession,
			}
			err := ts.StorageProvider.AddAuditLog(ctx, auditLog)
			require.NoError(t, err)

			pagination := &model.Pagination{
				Limit:  10,
				Offset: 0,
			}
			// Filter from 1 second ago to now+1
			fromTs := now - 1
			toTs := now + 1
			logs, _, err := ts.StorageProvider.ListAuditLogs(ctx, pagination, map[string]interface{}{
				"action":         uniqueAction,
				"from_timestamp": fromTs,
				"to_timestamp":   toTs,
			})
			require.NoError(t, err)
			assert.Equal(t, 1, len(logs))

			// Filter with future range should return no results
			futureTs := now + 3600
			logs, _, err = ts.StorageProvider.ListAuditLogs(ctx, pagination, map[string]interface{}{
				"action":         uniqueAction,
				"from_timestamp": futureTs,
			})
			require.NoError(t, err)
			assert.Equal(t, 0, len(logs))
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

		t.Run("should delete audit logs before created_at", func(t *testing.T) {
			uniqueAction := "cleanup_test_" + uuid.New().String()[:8]

			oldLog := &schemas.AuditLog{
				ActorID:      uuid.New().String(),
				ActorType:    constants.AuditActorTypeUser,
				ActorEmail:   "system@example.com",
				Action:       uniqueAction,
				CreatedAt:    time.Now().Add(-24 * time.Hour).Unix(),
				ResourceType: constants.AuditResourceTypeUser,
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

		t.Run("should preserve all fields in audit log round-trip", func(t *testing.T) {
			actorID := uuid.New().String()
			resourceID := uuid.New().String()
			uniqueAction := "roundtrip_" + uuid.New().String()[:8]

			auditLog := &schemas.AuditLog{
				ActorID:      actorID,
				ActorType:    constants.AuditActorTypeAdmin,
				ActorEmail:   "admin@test.com",
				Action:       uniqueAction,
				ResourceType: constants.AuditResourceTypeEmailTemplate,
				ResourceID:   resourceID,
				IPAddress:    "192.168.1.1",
				UserAgent:    "Mozilla/5.0 Test",
				Metadata:     `{"key":"value"}`,
			}
			err := ts.StorageProvider.AddAuditLog(ctx, auditLog)
			require.NoError(t, err)

			pagination := &model.Pagination{Limit: 10, Offset: 0}
			logs, _, err := ts.StorageProvider.ListAuditLogs(ctx, pagination, map[string]interface{}{
				"action": uniqueAction,
			})
			require.NoError(t, err)
			require.Len(t, logs, 1)

			got := logs[0]
			assert.Equal(t, actorID, got.ActorID)
			assert.Equal(t, constants.AuditActorTypeAdmin, got.ActorType)
			assert.Equal(t, "admin@test.com", got.ActorEmail)
			assert.Equal(t, uniqueAction, got.Action)
			assert.Equal(t, constants.AuditResourceTypeEmailTemplate, got.ResourceType)
			assert.Equal(t, resourceID, got.ResourceID)
			assert.Equal(t, "192.168.1.1", got.IPAddress)
			assert.Equal(t, "Mozilla/5.0 Test", got.UserAgent)
			assert.Equal(t, `{"key":"value"}`, got.Metadata)
			assert.NotZero(t, got.CreatedAt)
		})
	})
}

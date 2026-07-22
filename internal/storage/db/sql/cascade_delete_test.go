package sql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestDeleteWebhookAtomicRollback proves the cascade delete in DeleteWebhook runs
// inside a single transaction: when the second step (deleting webhook_logs) fails,
// the first step (deleting the webhook row) is rolled back rather than committed
// on its own. It exercises the same p.db.WithContext(ctx).Transaction pattern
// shared by DeleteClient, DeleteOrganization and DeleteUser — before the fix the
// first delete auto-committed and left the row lost when the second step errored.
func TestDeleteWebhookAtomicRollback(t *testing.T) {
	for _, dbType := range sqlMigrationTestDBTypes() {
		t.Run(dbType, func(t *testing.T) {
			cfg := sqlMigrationTestConfig(t, dbType)
			p, err := NewProvider(cfg, sqlTestDeps(t))
			require.NoError(t, err)
			defer func() { _ = p.Close() }()

			ctx := context.Background()

			wh, err := p.AddWebhook(ctx, &schemas.Webhook{
				EventName: "user.access.granted",
				EndPoint:  "https://example.com/hook",
			})
			require.NoError(t, err)

			// Force the SECOND cascade step to fail deterministically by removing
			// the webhook_logs table. With no transaction the first delete would
			// already be committed and the webhook row lost.
			require.NoError(t, p.db.Migrator().DropTable(&schemas.WebhookLog{}))

			err = p.DeleteWebhook(ctx, wh)
			require.Error(t, err, "delete must fail when the second cascade step errors")

			// Atomicity: the webhook row survived because the whole cascade rolled back.
			got, err := p.GetWebhookByID(ctx, wh.ID)
			require.NoError(t, err, "webhook must still exist after the cascade rolled back")
			require.NotNil(t, got)
			assert.Equal(t, wh.ID, got.ID)
		})
	}
}

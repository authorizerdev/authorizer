package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func webhookLogsTest(t *testing.T, s TestSetup) {
	time.Sleep(30 * time.Second) // add sleep for webhooklogs to get generated as they are async
	t.Helper()
	t.Run("should get webhook logs", func(t *testing.T) {
		req, ctx := createContext(s)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.NoError(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		webhookLogs, err := resolvers.WebhookLogsResolver(ctx, nil)
		assert.NoError(t, err)
		assert.Greater(t, len(webhookLogs.WebhookLogs), 1)

		webhooks, err := resolvers.WebhooksResolver(ctx, &model.PaginatedInput{
			Pagination: &model.PaginationInput{
				Limit: refs.NewInt64Ref(20),
			},
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, webhooks)
		for _, w := range webhooks.Webhooks {
			t.Run(fmt.Sprintf("should get webhook for webhook_id:%s", w.ID), func(t *testing.T) {
				webhookLogs, err := resolvers.WebhookLogsResolver(ctx, &model.ListWebhookLogRequest{
					WebhookID: &w.ID,
				})
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, len(webhookLogs.WebhookLogs), 1, refs.StringValue(w.EventName))
				for _, wl := range webhookLogs.WebhookLogs {
					assert.Equal(t, refs.StringValue(wl.WebhookID), w.ID)
				}
			})
		}
	})
}

package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func webhookLogsTest(t *testing.T, s TestSetup) {
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

		webhooks, err := resolvers.WebhooksResolver(ctx, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, webhooks)

		for _, w := range webhooks.Webhooks {
			t.Run(fmt.Sprintf("should get webhook for webhook_id:%s", w.ID), func(t *testing.T) {
				webhookLogs, err := resolvers.WebhookLogsResolver(ctx, &model.ListWebhookLogRequest{
					WebhookID: &w.ID,
				})
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, len(webhookLogs.WebhookLogs), 1)
				for _, wl := range webhookLogs.WebhookLogs {
					assert.Equal(t, refs.StringValue(wl.WebhookID), w.ID)
				}
			})
		}
	})
}

package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/utils"
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

		webhooks, err := resolvers.WebhooksResolver(ctx, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, webhooks)

		webhookLogs, err := resolvers.WebhookLogsResolver(ctx, nil)
		assert.NoError(t, err)
		assert.Greater(t, len(webhookLogs.WebhookLogs), 1)

		for _, w := range webhooks.Webhooks {
			webhookLogs, err := resolvers.WebhookLogsResolver(ctx, &model.ListWebhookLogRequest{
				WebhookID: &w.ID,
			})
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(webhookLogs.WebhookLogs), 1)
			for _, wl := range webhookLogs.WebhookLogs {
				assert.Equal(t, utils.StringValue(wl.WebhookID), w.ID)
			}
		}
	})
}

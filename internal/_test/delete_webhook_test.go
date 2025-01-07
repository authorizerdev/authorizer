package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func deleteWebhookTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run("should delete webhook", func(t *testing.T) {
		req, ctx := createContext(s)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.NoError(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		// get all webhooks
		webhooks, err := db.Provider.ListWebhook(ctx, &model.Pagination{
			Limit:  20,
			Page:   1,
			Offset: 0,
		})
		assert.NoError(t, err)

		for _, w := range webhooks.Webhooks {
			res, err := resolvers.DeleteWebhookResolver(ctx, model.WebhookRequest{
				ID: w.ID,
			})

			assert.NoError(t, err)
			assert.NotNil(t, res)
			assert.NotEmpty(t, res.Message)
		}

		webhooks, err = db.Provider.ListWebhook(ctx, &model.Pagination{
			Limit:  20,
			Page:   1,
			Offset: 0,
		})
		assert.NoError(t, err)
		assert.Len(t, webhooks.Webhooks, 0)
		webhookLogs, err := db.Provider.ListWebhookLogs(ctx, &model.Pagination{
			Limit:  100,
			Page:   1,
			Offset: 0,
		}, "")
		assert.NoError(t, err)
		assert.Len(t, webhookLogs.WebhookLogs, 0)
	})
}

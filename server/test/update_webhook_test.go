package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func updateWebhookTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run("should update webhook", func(t *testing.T) {
		req, ctx := createContext(s)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.NoError(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		// get webhook
		webhooks, err := db.Provider.GetWebhookByEventName(ctx, constants.UserDeletedWebhookEvent)
		assert.NoError(t, err)
		assert.NotNil(t, webhooks)
		assert.Greater(t, len(webhooks), 0)
		for _, webhook := range webhooks {
			// it should completely replace headers
			webhook.Headers = map[string]interface{}{
				"x-new-test": "test",
			}
			res, err := resolvers.UpdateWebhookResolver(ctx, model.UpdateWebhookRequest{
				ID:       webhook.ID,
				Headers:  webhook.Headers,
				Enabled:  refs.NewBoolRef(false),
				Endpoint: refs.NewStringRef("https://sometest.com"),
			})
			assert.NoError(t, err)
			assert.NotEmpty(t, res)
			assert.NotEmpty(t, res.Message)
		}

		updatedWebhooks, err := db.Provider.GetWebhookByEventName(ctx, constants.UserDeletedWebhookEvent)
		assert.NoError(t, err)
		assert.NotNil(t, updatedWebhooks)
		for _, updatedWebhook := range updatedWebhooks {
			assert.Contains(t, refs.StringValue(updatedWebhook.EventName), constants.UserDeletedWebhookEvent)
			assert.Len(t, updatedWebhook.Headers, 1)
			assert.False(t, refs.BoolValue(updatedWebhook.Enabled))
			for key, val := range updatedWebhook.Headers {
				assert.Equal(t, "x-new-test", key)
				assert.Equal(t, "test", val)
			}
			assert.Equal(t, "https://sometest.com", refs.StringValue(updatedWebhook.Endpoint))
			res, err := resolvers.UpdateWebhookResolver(ctx, model.UpdateWebhookRequest{
				ID:       updatedWebhook.ID,
				Headers:  updatedWebhook.Headers,
				Enabled:  refs.NewBoolRef(true),
				Endpoint: refs.NewStringRef(s.TestInfo.WebhookEndpoint),
			})
			assert.NoError(t, err)
			assert.NotEmpty(t, res)
			assert.NotEmpty(t, res.Message)
		}
	})
}

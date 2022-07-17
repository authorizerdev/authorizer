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
		webhook, err := db.Provider.GetWebhookByEventName(ctx, constants.UserDeletedWebhookEvent)
		assert.NoError(t, err)
		assert.NotNil(t, webhook)
		// it should completely replace headers
		webhook.Headers = map[string]interface{}{
			"x-new-test": "test",
		}

		res, err := resolvers.UpdateWebhookResolver(ctx, model.UpdateWebhookRequest{
			ID:      webhook.ID,
			Headers: webhook.Headers,
			Enabled: refs.NewBoolRef(false),
		})

		assert.NoError(t, err)
		assert.NotEmpty(t, res)
		assert.NotEmpty(t, res.Message)

		updatedWebhook, err := db.Provider.GetWebhookByEventName(ctx, constants.UserDeletedWebhookEvent)
		assert.NoError(t, err)
		assert.NotNil(t, updatedWebhook)
		assert.Equal(t, webhook.ID, updatedWebhook.ID)
		assert.Equal(t, refs.StringValue(webhook.EventName), refs.StringValue(updatedWebhook.EventName))
		assert.Equal(t, refs.StringValue(webhook.Endpoint), refs.StringValue(updatedWebhook.Endpoint))
		assert.Len(t, updatedWebhook.Headers, 1)
		assert.False(t, refs.BoolValue(updatedWebhook.Enabled))
		for key, val := range updatedWebhook.Headers {
			assert.Equal(t, val, webhook.Headers[key])
		}

		res, err = resolvers.UpdateWebhookResolver(ctx, model.UpdateWebhookRequest{
			ID:      webhook.ID,
			Headers: webhook.Headers,
			Enabled: refs.NewBoolRef(true),
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, res)
		assert.NotEmpty(t, res.Message)
	})
}

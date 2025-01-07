package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func webhookTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run("should get webhook", func(t *testing.T) {
		req, ctx := createContext(s)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.NoError(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		// get webhook by event name
		webhooks, err := db.Provider.GetWebhookByEventName(ctx, constants.UserCreatedWebhookEvent)
		assert.NoError(t, err)
		assert.NotNil(t, webhooks)
		assert.GreaterOrEqual(t, len(webhooks), 2)
		for _, webhook := range webhooks {
			res, err := resolvers.WebhookResolver(ctx, model.WebhookRequest{
				ID: webhook.ID,
			})
			assert.NoError(t, err)
			assert.Equal(t, res.ID, webhook.ID)
			assert.Equal(t, refs.StringValue(res.Endpoint), refs.StringValue(webhook.Endpoint))
			// assert.Equal(t, refs.StringValue(res.EventName), refs.StringValue(webhook.EventName))
			assert.Equal(t, refs.BoolValue(res.Enabled), refs.BoolValue(webhook.Enabled))
			assert.Len(t, res.Headers, len(webhook.Headers))
		}
	})
}

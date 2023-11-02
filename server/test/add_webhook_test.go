package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func addWebhookTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run("should add webhook", func(t *testing.T) {
		req, ctx := createContext(s)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.NoError(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		for _, eventType := range s.TestInfo.TestWebhookEventTypes {
			webhook, err := resolvers.AddWebhookResolver(ctx, model.AddWebhookRequest{
				EventName: eventType,
				Endpoint:  s.TestInfo.WebhookEndpoint,
				Enabled:   true,
				Headers: map[string]interface{}{
					"x-test": "foo",
				},
			})
			assert.NoError(t, err)
			assert.NotNil(t, webhook)
			assert.NotEmpty(t, webhook.Message)
		}
		time.Sleep(2 * time.Second)
		// Allow setting multiple webhooks for same event
		for _, eventType := range s.TestInfo.TestWebhookEventTypes {
			webhook, err := resolvers.AddWebhookResolver(ctx, model.AddWebhookRequest{
				EventName:        eventType,
				Endpoint:         s.TestInfo.WebhookEndpoint,
				Enabled:          true,
				EventDescription: refs.NewStringRef(eventType + "-2"),
				Headers: map[string]interface{}{
					"x-test": "foo",
				},
			})
			assert.NoError(t, err)
			assert.NotNil(t, webhook)
			assert.NotEmpty(t, webhook.Message)
		}
	})
}

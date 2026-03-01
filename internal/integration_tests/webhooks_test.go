package integration_tests

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebhooks tests the _webhooks list and _webhook single queries
func TestWebhooks(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	t.Run("should fail list webhooks without admin auth", func(t *testing.T) {
		req.Header.Set("Cookie", "")
		res, err := ts.GraphQLProvider.Webhooks(ctx, &model.PaginatedRequest{})
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should list webhooks with admin auth", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		res, err := ts.GraphQLProvider.Webhooks(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res.Pagination)
	})

	t.Run("should add and get single webhook", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		// Add a webhook
		addRes, err := ts.GraphQLProvider.AddWebhook(ctx, &model.AddWebhookRequest{
			EventName:        constants.UserLoginWebhookEvent,
			Endpoint:         "https://example.com/webhook",
			Enabled:          true,
			EventDescription: refs.NewStringRef("Test webhook"),
		})
		require.NoError(t, err)
		assert.NotNil(t, addRes)

		// List webhooks to get the ID
		webhooks, err := ts.GraphQLProvider.Webhooks(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(webhooks.Webhooks), 1)

		webhookID := webhooks.Webhooks[0].ID

		// Get single webhook
		webhook, err := ts.GraphQLProvider.Webhook(ctx, &model.WebhookRequest{
			ID: webhookID,
		})
		require.NoError(t, err)
		assert.NotNil(t, webhook)
		assert.Equal(t, webhookID, webhook.ID)
	})

	t.Run("should fail get webhook with invalid ID", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		webhook, err := ts.GraphQLProvider.Webhook(ctx, &model.WebhookRequest{
			ID: uuid.New().String(),
		})
		assert.Error(t, err)
		assert.Nil(t, webhook)
	})
}

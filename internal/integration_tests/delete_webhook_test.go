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

// TestDeleteWebhookTest tests the delete webhook functionality by the admin
func TestDeleteWebhookTest(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "delete_webhook_user_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)
	require.NotNil(t, signupRes)
	require.NotNil(t, signupRes.User)

	// First add a webhook to delete
	h, err := crypto.EncryptPassword(cfg.AdminSecret)
	assert.Nil(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	eventNameToDelete := constants.UserCreatedWebhookEvent
	eventNameToPersist := constants.UserLoginWebhookEvent

	addedWebhook, err := ts.GraphQLProvider.AddWebhook(ctx, &model.AddWebhookRequest{
		EventName:        eventNameToDelete,
		EventDescription: refs.NewStringRef("webhook to be deleted"),
		Endpoint:         "http://webhook-to-delete.com",
		Enabled:          true,
		Headers: map[string]any{
			"Content-Type": "application/json",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, addedWebhook)

	// Get the webhook to delete
	webhooks, err := ts.StorageProvider.GetWebhookByEventName(ctx, eventNameToDelete)
	require.NoError(t, err)
	require.NotEmpty(t, webhooks)
	webhookID := webhooks[0].ID

	// Create another webhook that won't be deleted (to verify we're not deleting all webhooks)
	persistentWebhook, err := ts.GraphQLProvider.AddWebhook(ctx, &model.AddWebhookRequest{
		EventName:        eventNameToPersist,
		EventDescription: refs.NewStringRef("webhook that should persist"),
		Endpoint:         "http://persistent-webhook.com",
		Enabled:          true,
	})
	require.NoError(t, err)
	assert.NotNil(t, persistentWebhook)

	t.Run("should fail without admin cookie", func(t *testing.T) {
		// Remove admin cookie
		req.Header.Del("Cookie")

		resp, err := ts.GraphQLProvider.DeleteWebhook(ctx, &model.WebhookRequest{
			ID: webhookID,
		})
		require.Error(t, err)
		require.Nil(t, resp)
	})

	// Re-add the admin cookie for the rest of the tests
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	t.Run("should fail with invalid webhook ID", func(t *testing.T) {
		resp, err := ts.GraphQLProvider.DeleteWebhook(ctx, &model.WebhookRequest{
			ID: uuid.NewString(),
		})
		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("should fail with blank webhook ID", func(t *testing.T) {
		resp, err := ts.GraphQLProvider.DeleteWebhook(ctx, &model.WebhookRequest{
			ID: "",
		})
		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("should delete webhook successfully", func(t *testing.T) {
		// Verify webhook exists before deletion
		webhookBeforeDelete, err := ts.StorageProvider.GetWebhookByID(ctx, webhookID)
		require.NoError(t, err)
		require.NotNil(t, webhookBeforeDelete)

		// Delete the webhook
		resp, err := ts.GraphQLProvider.DeleteWebhook(ctx, &model.WebhookRequest{
			ID: webhookID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "successfully")

		// Verify webhook was deleted
		webhookAfterDelete, err := ts.StorageProvider.GetWebhookByID(ctx, webhookID)
		require.Error(t, err) // Should error because webhook no longer exists
		require.Nil(t, webhookAfterDelete)

		// Verify other webhooks still exist
		persistentWebhooks, err := ts.StorageProvider.GetWebhookByEventName(ctx, eventNameToPersist)
		require.NoError(t, err)
		require.NotEmpty(t, persistentWebhooks)
	})

	t.Run("should fail to delete already deleted webhook", func(t *testing.T) {
		// Try to delete the same webhook again
		resp, err := ts.GraphQLProvider.DeleteWebhook(ctx, &model.WebhookRequest{
			ID: webhookID,
		})
		require.Error(t, err)
		require.Nil(t, resp)
	})
}

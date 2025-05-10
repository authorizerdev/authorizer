package integration_tests

import (
	"fmt"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

// TestUpdateWebhookTest tests the update webhook functionality by the admin
func TestUpdateWebhookTest(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "update_webhook_user_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpInput{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)
	require.NotNil(t, signupRes)
	require.NotNil(t, signupRes.User)

	// First add a webhook to update
	h, err := crypto.EncryptPassword(cfg.AdminSecret)
	assert.Nil(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	// Create a webhook with original event name
	addedWebhook, err := ts.GraphQLProvider.AddWebhook(ctx, &model.AddWebhookRequest{
		EventName:        constants.UserCreatedWebhookEvent,
		EventDescription: refs.NewStringRef("original description"),
		Endpoint:         "http://original-endpoint.com",
		Enabled:          false,
		Headers: map[string]any{
			"Content-Type": "application/json",
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, addedWebhook)

	// Get the webhook to update
	webhooks, err := ts.StorageProvider.GetWebhookByEventName(ctx, constants.UserCreatedWebhookEvent)
	require.NoError(t, err)
	require.NotEmpty(t, webhooks)
	webhookID := webhooks[0].ID

	t.Run("should fail without admin cookie", func(t *testing.T) {
		// Remove admin cookie
		req.Header.Del("Cookie")

		updatedWebhook, err := ts.GraphQLProvider.UpdateWebhook(ctx, &model.UpdateWebhookRequest{
			ID:               webhookID,
			EventName:        refs.NewStringRef(constants.UserCreatedWebhookEvent),
			EventDescription: refs.NewStringRef("updated description"),
			Endpoint:         refs.NewStringRef("http://updated-endpoint.com"),
			Enabled:          refs.NewBoolRef(true),
			Headers: map[string]any{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token",
			},
		})
		require.Error(t, err)
		require.Nil(t, updatedWebhook)
	})

	// Re-add the admin cookie for the rest of the tests
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	t.Run("should fail with invalid webhook ID", func(t *testing.T) {
		updatedWebhook, err := ts.GraphQLProvider.UpdateWebhook(ctx, &model.UpdateWebhookRequest{
			ID:               uuid.NewString(),
			EventName:        refs.NewStringRef(constants.UserCreatedWebhookEvent),
			EventDescription: refs.NewStringRef("updated description"),
			Endpoint:         refs.NewStringRef("http://updated-endpoint.com"),
			Enabled:          refs.NewBoolRef(true),
		})
		require.Error(t, err)
		require.Nil(t, updatedWebhook)
	})

	t.Run("should fail with blank webhook ID", func(t *testing.T) {
		updatedWebhook, err := ts.GraphQLProvider.UpdateWebhook(ctx, &model.UpdateWebhookRequest{
			ID:               "",
			EventName:        refs.NewStringRef(constants.UserCreatedWebhookEvent),
			EventDescription: refs.NewStringRef("updated description"),
			Endpoint:         refs.NewStringRef("http://updated-endpoint.com"),
			Enabled:          refs.NewBoolRef(true),
		})
		require.Error(t, err)
		require.Nil(t, updatedWebhook)
	})

	t.Run("should fail with invalid event name", func(t *testing.T) {
		updatedWebhook, err := ts.GraphQLProvider.UpdateWebhook(ctx, &model.UpdateWebhookRequest{
			ID:               webhookID,
			EventName:        refs.NewStringRef("invalid_event_name"),
			EventDescription: refs.NewStringRef("updated description"),
			Endpoint:         refs.NewStringRef("http://updated-endpoint.com"),
			Enabled:          refs.NewBoolRef(true),
		})
		require.Error(t, err)
		require.Nil(t, updatedWebhook)
	})

	t.Run("should fail with blank endpoint", func(t *testing.T) {
		updatedWebhook, err := ts.GraphQLProvider.UpdateWebhook(ctx, &model.UpdateWebhookRequest{
			ID:               webhookID,
			EventName:        refs.NewStringRef(constants.UserCreatedWebhookEvent),
			EventDescription: refs.NewStringRef("updated description"),
			Endpoint:         refs.NewStringRef(""),
			Enabled:          refs.NewBoolRef(true),
		})
		require.Error(t, err)
		require.Nil(t, updatedWebhook)
	})

	t.Run("should update webhook successfully", func(t *testing.T) {
		// Use a different event name for the update to avoid conflicts
		newEventName := constants.UserLoginWebhookEvent
		updatedEndpoint := "http://updated-endpoint.com"

		updatedWebhook, err := ts.GraphQLProvider.UpdateWebhook(ctx, &model.UpdateWebhookRequest{
			ID:               webhookID,
			EventName:        refs.NewStringRef(newEventName),
			EventDescription: refs.NewStringRef("updated description"),
			Endpoint:         refs.NewStringRef(updatedEndpoint),
			Enabled:          refs.NewBoolRef(true),
			Headers: map[string]any{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token",
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, updatedWebhook)

		// Verify the webhook was updated correctly
		updatedWebhooks, err := ts.StorageProvider.GetWebhookByID(ctx, webhookID)
		require.NoError(t, err)
		assert.NotNil(t, updatedWebhooks)

		// Use the same event name that we provided in the update request
		assert.Contains(t, updatedWebhooks.EventName, newEventName)
		assert.Equal(t, "updated description", updatedWebhooks.EventDescription)
		assert.Equal(t, updatedEndpoint, updatedWebhooks.EndPoint)
		assert.Equal(t, true, updatedWebhooks.Enabled)
	})

	t.Run("should partially update webhook", func(t *testing.T) {
		// First get the current webhook state to verify what fields stay unchanged
		currentWebhook, err := ts.StorageProvider.GetWebhookByID(ctx, webhookID)
		require.NoError(t, err)

		// Store the current values for later verification
		currentEventName := strings.Split(currentWebhook.EventName, "-")[0]
		currentEndpoint := currentWebhook.EndPoint

		// Update only the description and enabled status
		updatedWebhook, err := ts.GraphQLProvider.UpdateWebhook(ctx, &model.UpdateWebhookRequest{
			ID:               webhookID,
			EventDescription: refs.NewStringRef("new partial description"),
			Enabled:          refs.NewBoolRef(false),
			// Don't update event name or endpoint
		})
		require.NoError(t, err)
		assert.NotNil(t, updatedWebhook)

		// Verify only specified fields were updated
		afterPartialUpdate, err := ts.StorageProvider.GetWebhookByID(ctx, webhookID)
		require.NoError(t, err)
		assert.NotNil(t, afterPartialUpdate)

		// The event name should remain the same as before
		assert.Contains(t, afterPartialUpdate.EventName, currentEventName)

		// Description should be updated
		assert.Equal(t, "new partial description", afterPartialUpdate.EventDescription)

		// Endpoint should remain unchanged
		assert.Equal(t, currentEndpoint, afterPartialUpdate.EndPoint)

		// Enabled should be updated
		assert.Equal(t, false, afterPartialUpdate.Enabled)
	})
}

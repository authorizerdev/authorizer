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
	"testing"
)

// TestAddWebhookTest tests the add webhook functionality by the admin
func TestAddWebhookTest(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "add_webhook_user_test_" + uuid.New().String() + "@authorizer.dev"
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

	t.Run("should fail without admin cookie", func(t *testing.T) {
		addedWebhook, err := ts.GraphQLProvider.AddWebhook(ctx, &model.AddWebhookRequest{
			EventName:        "test",
			EventDescription: refs.NewStringRef("test"),
			Endpoint:         "test",
			Enabled:          false,
			Headers: map[string]any{
				"test": "test",
			},
		})
		require.Error(t, err)
		require.Nil(t, addedWebhook)
	})

	t.Run("should fail with blank event name", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		addedWebhook, err := ts.GraphQLProvider.AddWebhook(ctx, &model.AddWebhookRequest{
			EventName:        "",
			EventDescription: refs.NewStringRef("test"),
			Endpoint:         "test",
			Enabled:          false,
			Headers: map[string]any{
				"test": "test",
			},
		})
		require.Error(t, err)
		require.Nil(t, addedWebhook)
	})

	t.Run("should fail with blank endpoint", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		addedWebhook, err := ts.GraphQLProvider.AddWebhook(ctx, &model.AddWebhookRequest{
			EventName:        "test",
			EventDescription: refs.NewStringRef("test"),
			Endpoint:         "",
			Enabled:          false,
			Headers: map[string]any{
				"test": "test",
			},
		})
		require.Error(t, err)
		require.Nil(t, addedWebhook)
	})

	t.Run("should add webhook", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		addedWebhook, err := ts.GraphQLProvider.AddWebhook(ctx, &model.AddWebhookRequest{
			EventName:        constants.UserCreatedWebhookEvent,
			EventDescription: refs.NewStringRef("test"),
			Endpoint:         "test",
			Enabled:          false,
			Headers: map[string]any{
				"test": "test",
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, addedWebhook)

		res, err := ts.StorageProvider.GetWebhookByEventName(ctx, constants.UserCreatedWebhookEvent)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, 1, len(res))
		assert.Equal(t, "test", res[0].EventDescription)
		assert.Equal(t, "test", res[0].EndPoint)
		assert.Equal(t, false, res[0].Enabled)
		assert.Equal(t, "{\"test\":\"test\"}", res[0].Headers)
		assert.NotNil(t, res[0].ID)
		assert.NotNil(t, res[0].CreatedAt)
		assert.NotNil(t, res[0].UpdatedAt)
		assert.NotNil(t, res[0].Key)
	})
}

package integration_tests

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebhookLogs tests the _webhook_logs query
func TestWebhookLogs(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	t.Run("should fail without admin auth", func(t *testing.T) {
		req.Header.Set("Cookie", "")
		res, err := ts.GraphQLProvider.WebhookLogs(ctx, &model.ListWebhookLogRequest{})
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should list webhook logs with admin auth", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		res, err := ts.GraphQLProvider.WebhookLogs(ctx, &model.ListWebhookLogRequest{})
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res.Pagination)
	})
}

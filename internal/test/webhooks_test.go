package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func webhooksTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run("should get webhooks", func(t *testing.T) {
		req, ctx := createContext(s)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.NoError(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		webhooks, err := resolvers.WebhooksResolver(ctx, &model.PaginatedInput{
			Pagination: &model.PaginationInput{
				Limit: refs.NewInt64Ref(20),
			},
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, webhooks)
		assert.GreaterOrEqual(t, len(webhooks.Webhooks), len(s.TestInfo.TestWebhookEventTypes)*2)
	})
}

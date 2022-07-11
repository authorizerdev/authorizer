package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/resolvers"
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

		webhooks, err := resolvers.WebhooksResolver(ctx, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, webhooks)
		assert.Len(t, webhooks.Webhooks, len(s.TestInfo.TestEventTypes))
	})
}

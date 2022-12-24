package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func testEndpointTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run("should test endpoint", func(t *testing.T) {
		req, ctx := createContext(s)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.NoError(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		res, err := resolvers.TestEndpointResolver(ctx, model.TestEndpointRequest{
			Endpoint:  s.TestInfo.WebhookEndpoint,
			EventName: constants.UserLoginWebhookEvent,
			Headers: map[string]interface{}{
				"x-test": "test",
			},
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.GreaterOrEqual(t, *res.HTTPStatus, int64(200))
		assert.NotEmpty(t, res.Response)
	})
}

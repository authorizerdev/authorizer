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

func adminSessionTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get admin session`, func(t *testing.T) {
		req, ctx := createContext(s)
		_, err := resolvers.AdminSessionResolver(ctx)
		assert.NotNil(t, err)

		h, err := crypto.EncryptPassword(memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminCookieName), h))
		_, err = resolvers.AdminSessionResolver(ctx)

		assert.Nil(t, err)
	})
}

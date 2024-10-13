package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func adminSessionTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get admin session`, func(t *testing.T) {
		req, ctx := createContext(s)
		_, err := resolvers.AdminSessionResolver(ctx)
		assert.NotNil(t, err)

		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)

		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		_, err = resolvers.AdminSessionResolver(ctx)

		assert.Nil(t, err)
	})
}

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

func envTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get envs`, func(t *testing.T) {
		req, ctx := createContext(s)
		_, err := resolvers.EnvResolver(ctx)
		assert.NotNil(t, err)

		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)

		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		res, err := resolvers.EnvResolver(ctx)
		assert.Nil(t, err)
		assert.Equal(t, *res.AdminSecret, adminSecret)
	})
}

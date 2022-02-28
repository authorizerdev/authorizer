package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func envTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get envs`, func(t *testing.T) {
		req, ctx := createContext(s)
		_, err := resolvers.EnvResolver(ctx)
		assert.NotNil(t, err)

		h, err := crypto.EncryptPassword(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminCookieName), h))
		res, err := resolvers.EnvResolver(ctx)

		assert.Nil(t, err)
		assert.Equal(t, *res.AdminSecret, envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
	})
}

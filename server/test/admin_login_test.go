package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func adminLoginTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should complete admin login`, func(t *testing.T) {
		_, ctx := createContext(s)
		_, err := resolvers.AdminLoginResolver(ctx, model.AdminLoginInput{
			AdminSecret: "admin_test",
		})

		assert.NotNil(t, err)

		_, err = resolvers.AdminLoginResolver(ctx, model.AdminLoginInput{
			AdminSecret: envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret),
		})

		assert.Nil(t, err)
	})
}

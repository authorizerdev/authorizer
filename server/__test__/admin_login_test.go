package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func adminLoginTests(s TestSetup, t *testing.T) {
	t.Run(`should complete admin login`, func(t *testing.T) {
		_, ctx := createContext(s)
		_, err := resolvers.AdminLoginResolver(ctx, model.AdminLoginInput{
			AdminSecret: "admin_test",
		})

		assert.NotNil(t, err)

		_, err = resolvers.AdminLoginResolver(ctx, model.AdminLoginInput{
			AdminSecret: constants.EnvData.ADMIN_SECRET,
		})

		assert.Nil(t, err)
	})
}

package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func aminLoginTests(s TestSetup, t *testing.T) {
	t.Run(`should complete admin login`, func(t *testing.T) {
		_, ctx := createContext(s)
		_, err := resolvers.AdminLoginResolver(ctx, model.AdminLoginInput{
			AdminSecret: "admin_test",
		})

		assert.NotNil(t, err)

		res, err := resolvers.AdminLoginResolver(ctx, model.AdminLoginInput{
			AdminSecret: "admin",
		})

		assert.Nil(t, err)
		assert.Greater(t, len(res.AccessToken), 0)
	})
}

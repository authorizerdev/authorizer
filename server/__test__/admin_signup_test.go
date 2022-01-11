package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func adminSignupTests(s TestSetup, t *testing.T) {
	t.Run(`should complete admin login`, func(t *testing.T) {
		_, ctx := createContext(s)
		_, err := resolvers.AdminSignupResolver(ctx, model.AdminSignupInput{
			AdminSecret: "admin",
		})

		assert.NotNil(t, err)
		// reset env for test to pass
		constants.EnvData.ADMIN_SECRET = ""

		_, err = resolvers.AdminSignupResolver(ctx, model.AdminSignupInput{
			AdminSecret: uuid.New().String(),
		})

		assert.Nil(t, err)
	})
}

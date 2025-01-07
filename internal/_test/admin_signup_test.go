package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func adminSignupTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should complete admin signup`, func(t *testing.T) {
		_, ctx := createContext(s)
		_, err := resolvers.AdminSignupResolver(ctx, model.AdminSignupInput{
			AdminSecret: "admin",
		})
		assert.NotNil(t, err)
		// reset env for test to pass
		err = memorystore.Provider.UpdateEnvVariable(constants.EnvKeyAdminSecret, "")
		assert.Nil(t, err)
		_, err = resolvers.AdminSignupResolver(ctx, model.AdminSignupInput{
			AdminSecret: "admin123",
		})
		assert.NoError(t, err)
	})
}

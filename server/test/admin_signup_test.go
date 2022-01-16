package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func adminSignupTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should complete admin login`, func(t *testing.T) {
		_, ctx := createContext(s)
		_, err := resolvers.AdminSignupResolver(ctx, model.AdminSignupInput{
			AdminSecret: "admin",
		})

		assert.NotNil(t, err)
		// reset env for test to pass
		envstore.EnvInMemoryStoreObj.UpdateEnvVariable(constants.EnvKeyAdminSecret, "")

		_, err = resolvers.AdminSignupResolver(ctx, model.AdminSignupInput{
			AdminSecret: uuid.New().String(),
		})

		assert.Nil(t, err)
	})
}

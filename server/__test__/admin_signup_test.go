package test

import (
	"log"
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
		_, err := resolvers.AdminSignupResolver(ctx, model.AdminLoginInput{
			AdminSecret: "admin",
		})
		log.Println("err", err)
		assert.NotNil(t, err)
		// reset env for test to pass
		constants.EnvData.ADMIN_SECRET = ""

		res, err := resolvers.AdminSignupResolver(ctx, model.AdminLoginInput{
			AdminSecret: uuid.New().String(),
		})

		assert.Nil(t, err)
		assert.Greater(t, len(res.AccessToken), 0)
	})
}

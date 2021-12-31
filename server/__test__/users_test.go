package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func usersTest(s TestSetup, t *testing.T) {
	t.Run(`should get users list with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "users." + s.TestInfo.Email
		resolvers.Signup(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		users, err := resolvers.Users(ctx)
		assert.NotNil(t, err, "unauthorized")

		req.Header.Add("x-authorizer-admin-secret", constants.EnvData.ADMIN_SECRET)
		users, err = resolvers.Users(ctx)
		assert.Nil(t, err)
		rLen := len(users)
		assert.GreaterOrEqual(t, rLen, 1)

		cleanData(email)
	})
}

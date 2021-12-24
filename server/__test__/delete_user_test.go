package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func deleteUserTest(s TestSetup, t *testing.T) {
	t.Run(`should delete users with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "delete_user." + s.TestInfo.Email
		resolvers.Signup(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err := resolvers.DeleteUser(ctx, model.DeleteUserInput{
			Email: email,
		})
		assert.NotNil(t, err, "unauthorized")

		req.Header.Add("x-authorizer-admin-secret", constants.ADMIN_SECRET)
		_, err = resolvers.DeleteUser(ctx, model.DeleteUserInput{
			Email: email,
		})
		assert.Nil(t, err)
		cleanData(email)
	})
}

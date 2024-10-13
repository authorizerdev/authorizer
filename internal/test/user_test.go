package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func userTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get users list with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "user." + s.TestInfo.Email
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, res.User)

		userRes, err := resolvers.UserResolver(ctx, model.GetUserRequest{
			ID: &res.User.ID,
		})
		assert.Nil(t, userRes)
		assert.NotNil(t, err, "unauthorized")

		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		// Should throw error for invalid params
		userRes, err = resolvers.UserResolver(ctx, model.GetUserRequest{})
		assert.Nil(t, userRes)
		assert.NotNil(t, err, "invalid params, user id or email is required")
		// Should throw error for invalid params with empty id
		userRes, err = resolvers.UserResolver(ctx, model.GetUserRequest{
			ID: refs.NewStringRef("   "),
		})
		assert.Nil(t, userRes)
		assert.NotNil(t, err, "invalid params, user id or email is required")
		// Should throw error for invalid params with empty email
		userRes, err = resolvers.UserResolver(ctx, model.GetUserRequest{
			Email: refs.NewStringRef("   "),
		})
		assert.Nil(t, userRes)
		assert.NotNil(t, err, "invalid params, user id or email is required")
		// Should get user by id
		userRes, err = resolvers.UserResolver(ctx, model.GetUserRequest{
			ID: &res.User.ID,
		})
		assert.Nil(t, err)
		assert.Equal(t, res.User.ID, userRes.ID)
		assert.Equal(t, email, refs.StringValue(userRes.Email))
		// Should get user by email
		userRes, err = resolvers.UserResolver(ctx, model.GetUserRequest{
			Email: &email,
		})
		assert.Nil(t, err)
		assert.Equal(t, res.User.ID, userRes.ID)
		assert.Equal(t, email, refs.StringValue(userRes.Email))
		cleanData(email)
	})
}

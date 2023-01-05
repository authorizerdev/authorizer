package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func userTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get users list with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "user." + s.TestInfo.Email
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, res.User)

		userRes, err := resolvers.UserResolver(ctx, model.GetUserRequest{
			ID: res.User.ID,
		})
		assert.Nil(t, userRes)
		assert.NotNil(t, err, "unauthorized")

		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		userRes, err = resolvers.UserResolver(ctx, model.GetUserRequest{
			ID: res.User.ID,
		})
		assert.Nil(t, err)
		assert.Equal(t, res.User.ID, userRes.ID)
		assert.Equal(t, email, userRes.Email)

		cleanData(email)
	})
}

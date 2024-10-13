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

func deleteUserTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should delete users with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "delete_user." + s.TestInfo.Email
		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err := resolvers.DeleteUserResolver(ctx, model.DeleteUserInput{
			Email: email,
		})
		assert.NotNil(t, err, "unauthorized")
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)

		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		_, err = resolvers.DeleteUserResolver(ctx, model.DeleteUserInput{
			Email: email,
		})
		assert.Nil(t, err)
		cleanData(email)
	})
}

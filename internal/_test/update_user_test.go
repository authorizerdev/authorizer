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

func updateUserTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should update the user with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "update_user." + s.TestInfo.Email
		signupRes, _ := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		user := *signupRes.User

		adminRole := "supplier"
		userRole := "user"
		newRoles := []*string{&adminRole, &userRole}
		_, err := resolvers.UpdateUserResolver(ctx, model.UpdateUserInput{
			ID:    user.ID,
			Roles: newRoles,
		})
		assert.NotNil(t, err, "unauthorized")
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		_, err = resolvers.UpdateUserResolver(ctx, model.UpdateUserInput{
			ID:    user.ID,
			Roles: newRoles,
		})
		// supplier is not part of envs
		assert.Error(t, err)
		adminRole = "admin"
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyProtectedRoles, adminRole)
		newRoles = []*string{&adminRole, &userRole}
		_, err = resolvers.UpdateUserResolver(ctx, model.UpdateUserInput{
			ID:    user.ID,
			Roles: newRoles,
			AppData: map[string]interface{}{
				"test": "test",
			},
		})
		assert.Nil(t, err)
		// Get user and check if roles are updated
		users, err := resolvers.UsersResolver(ctx, nil)
		assert.Nil(t, err)
		for _, u := range users.Users {
			if u.ID == user.ID {
				assert.Equal(t, u.AppData["test"], "test")
			}
		}
		cleanData(email)
	})
}

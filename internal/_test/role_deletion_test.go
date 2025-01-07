package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/crypto"

	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
)

func RoleDeletionTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should complete role deletion`, func(t *testing.T) {
		// login as admin
		req, ctx := createContext(s)

		_, err := resolvers.AdminLoginResolver(ctx, model.AdminLoginInput{
			AdminSecret: "admin_test",
		})
		assert.NotNil(t, err)

		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)
		_, err = resolvers.AdminLoginResolver(ctx, model.AdminLoginInput{
			AdminSecret: adminSecret,
		})
		assert.Nil(t, err)

		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		// add new default role to get role, if not present in roles
		originalDefaultRoles, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyDefaultRoles)
		assert.Nil(t, err)
		originalDefaultRolesSlice := strings.Split(originalDefaultRoles, ",")

		data := model.UpdateEnvInput{
			DefaultRoles: append(originalDefaultRolesSlice, "abc"),
		}
		_, err = resolvers.UpdateEnvResolver(ctx, data)
		assert.Error(t, err)

		// add new role
		originalRoles, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyRoles)
		assert.Nil(t, err)
		originalRolesSlice := strings.Split(originalRoles, ",")
		roleToBeAdded := "abc"
		newRoles := append(originalRolesSlice, roleToBeAdded)
		data = model.UpdateEnvInput{
			Roles: newRoles,
		}
		_, err = resolvers.UpdateEnvResolver(ctx, data)
		assert.Nil(t, err)

		// register a user with all roles
		email := "update_user." + s.TestInfo.Email
		_, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
			Roles:           newRoles,
		})
		assert.Nil(t, err)

		regUserDetails, _ := resolvers.UserResolver(ctx, model.GetUserRequest{
			Email: refs.NewStringRef(email),
		})

		// update env by removing role "abc"
		var newRolesAfterDeletion []string
		for _, value := range newRoles {
			if value != roleToBeAdded {
				newRolesAfterDeletion = append(newRolesAfterDeletion, value)
			}
		}
		data = model.UpdateEnvInput{
			Roles: newRolesAfterDeletion,
		}
		_, err = resolvers.UpdateEnvResolver(ctx, data)
		assert.Nil(t, err)

		// check user if role still exist
		userDetails, err := resolvers.UserResolver(ctx, model.GetUserRequest{
			Email: refs.NewStringRef(email),
		})
		assert.Nil(t, err)
		assert.Equal(t, newRolesAfterDeletion, userDetails.Roles)
		assert.NotEqual(t, newRolesAfterDeletion, regUserDetails.Roles)
	})
}

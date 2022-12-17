package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func updateAllUsersTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run("Should update all users", func(t *testing.T) {
		_, ctx := createContext(s)

		users := []models.User{}
		for i := 0; i < 10; i++ {
			user := models.User{
				Email:         fmt.Sprintf("update_all_user_%d_%s", i, s.TestInfo.Email),
				SignupMethods: constants.AuthRecipeMethodBasicAuth,
				Roles:         "user",
			}
			users = append(users, user)
			u, err := db.Provider.AddUser(ctx, user)
			assert.NoError(t, err)
			assert.NotNil(t, u)
		}

		err := db.Provider.UpdateUsers(ctx, map[string]interface{}{
			"is_multi_factor_auth_enabled": true,
		}, nil)
		assert.NoError(t, err)

		listUsers, err := db.Provider.ListUsers(ctx, model.Pagination{
			Limit:  20,
			Offset: 0,
		})
		assert.NoError(t, err)

		for _, u := range listUsers.Users {
			assert.True(t, refs.BoolValue(u.IsMultiFactorAuthEnabled))
		}

		// // update few users
		updateIds := []string{listUsers.Users[0].ID, listUsers.Users[1].ID}
		err = db.Provider.UpdateUsers(ctx, map[string]interface{}{
			"is_multi_factor_auth_enabled": false,
		}, updateIds)
		assert.NoError(t, err)

		listUsers, err = db.Provider.ListUsers(ctx, model.Pagination{
			Limit:  20,
			Offset: 0,
		})
		for _, u := range listUsers.Users {
			if utils.StringSliceContains(updateIds, u.ID) {
				assert.False(t, refs.BoolValue(u.IsMultiFactorAuthEnabled))
			} else {
				assert.True(t, refs.BoolValue(u.IsMultiFactorAuthEnabled))
			}
		}
	})
}

package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func updateUserTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should update the user with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "update_user." + s.TestInfo.Email
		signupRes, _ := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
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

		h, err := utils.EncryptPassword(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminCookieName), h))
		_, err = resolvers.UpdateUserResolver(ctx, model.UpdateUserInput{
			ID:    user.ID,
			Roles: newRoles,
		})
		assert.Nil(t, err)
		cleanData(email)
	})
}

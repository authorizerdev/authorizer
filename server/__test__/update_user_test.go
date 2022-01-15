package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func updateUserTest(s TestSetup, t *testing.T) {
	t.Helper()
	t.Run(`should update the user with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "update_user." + s.TestInfo.Email
		signupRes, _ := resolvers.Signup(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		user := *signupRes.User
		adminRole := "admin"
		userRole := "user"
		newRoles := []*string{&adminRole, &userRole}
		_, err := resolvers.UpdateUser(ctx, model.UpdateUserInput{
			ID:    user.ID,
			Roles: newRoles,
		})
		assert.NotNil(t, err, "unauthorized")

		h, err := utils.HashPassword(constants.EnvData.ADMIN_SECRET)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.EnvData.ADMIN_COOKIE_NAME, h))
		_, err = resolvers.UpdateUser(ctx, model.UpdateUserInput{
			ID:    user.ID,
			Roles: newRoles,
		})
		assert.Nil(t, err)
		cleanData(email)
	})
}

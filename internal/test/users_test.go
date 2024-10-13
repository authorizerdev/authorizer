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

func usersTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get users list with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "users." + s.TestInfo.Email
		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		limit := int64(10)
		page := int64(1)
		pagination := &model.PaginatedInput{
			Pagination: &model.PaginationInput{
				Limit: &limit,
				Page:  &page,
			},
		}

		usersRes, err := resolvers.UsersResolver(ctx, pagination)
		assert.NotNil(t, err, "unauthorized")
		assert.Nil(t, usersRes)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		usersRes, err = resolvers.UsersResolver(ctx, pagination)
		assert.Nil(t, err)
		rLen := len(usersRes.Users)
		assert.GreaterOrEqual(t, rLen, 1)

		cleanData(email)
	})
}

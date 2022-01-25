package test

import (
	"fmt"
	"log"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func usersTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get users list with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "users." + s.TestInfo.Email
		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
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

		h, err := utils.EncryptPassword(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminCookieName), h))

		usersRes, err = resolvers.UsersResolver(ctx, pagination)
		assert.Nil(t, err)
		log.Println("=> userRes:", usersRes)
		rLen := len(usersRes.Users)
		assert.GreaterOrEqual(t, rLen, 1)

		cleanData(email)
	})
}

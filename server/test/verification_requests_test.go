package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func verificationRequestsTest(t *testing.T, s TestSetup) {
	t.Helper()

	t.Run(`should get verification requests with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "verification_requests." + s.TestInfo.Email
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		limit := int64(10)
		page := int64(1)
		pagination := &model.PaginatedInput{
			Pagination: &model.PaginationInput{
				Limit: &limit,
				Page:  &page,
			},
		}

		requests, err := resolvers.VerificationRequestsResolver(ctx, pagination)
		assert.NotNil(t, err, "unauthorized")
		assert.Nil(t, requests)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)

		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		requests, err = resolvers.VerificationRequestsResolver(ctx, pagination)
		assert.Nil(t, err)
		rLen := len(requests.VerificationRequests)
		assert.GreaterOrEqual(t, rLen, 1)

		cleanData(email)
	})
}

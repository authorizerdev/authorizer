package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func verificationRequestsTest(s TestSetup, t *testing.T) {
	t.Run(`should get verification requests with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)

		email := "verification_requests." + s.TestInfo.Email
		resolvers.Signup(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		requests, err := resolvers.VerificationRequests(ctx)
		assert.NotNil(t, err, "unauthorizer")

		req.Header.Add("x-authorizer-admin-secret", constants.EnvData.ADMIN_SECRET)
		requests, err = resolvers.VerificationRequests(ctx)

		assert.Nil(t, err)
		rLen := len(requests)
		assert.GreaterOrEqual(t, rLen, 1)

		cleanData(email)
	})
}

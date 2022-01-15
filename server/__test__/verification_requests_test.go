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

func verificationRequestsTest(s TestSetup, t *testing.T) {
	t.Helper()
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

		h, err := utils.HashPassword(constants.EnvData.ADMIN_SECRET)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.EnvData.ADMIN_COOKIE_NAME, h))
		requests, err = resolvers.VerificationRequests(ctx)

		assert.Nil(t, err)
		rLen := len(requests)
		assert.GreaterOrEqual(t, rLen, 1)

		cleanData(email)
	})
}

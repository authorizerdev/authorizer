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

func verificationRequestsTest(t *testing.T, s TestSetup) {
	t.Helper()

	t.Run(`should get verification requests with admin secret only`, func(t *testing.T) {
		req, ctx := createContext(s)

		email := "verification_requests." + s.TestInfo.Email
		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		requests, err := resolvers.VerificationRequestsResolver(ctx)
		assert.NotNil(t, err, "unauthorized")

		h, err := utils.EncryptPassword(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminCookieName), h))
		requests, err = resolvers.VerificationRequestsResolver(ctx)

		assert.Nil(t, err)
		rLen := len(requests)
		assert.GreaterOrEqual(t, rLen, 1)

		cleanData(email)
	})
}

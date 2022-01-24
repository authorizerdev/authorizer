package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func profileTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get profile only with token`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "profile." + s.TestInfo.Email

		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err := resolvers.ProfileResolver(ctx)
		assert.NotNil(t, err, "unauthorized")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(email, constants.VerificationTypeBasicAuthSignup)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		token := *verifyRes.AccessToken
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".access_token", token))
		profileRes, err := resolvers.ProfileResolver(ctx)
		assert.Nil(t, err)

		newEmail := *&profileRes.Email
		assert.Equal(t, email, newEmail, "emails should be equal")

		cleanData(email)
	})
}

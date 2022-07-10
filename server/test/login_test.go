package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func loginTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should login`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "login." + s.TestInfo.Email
		_, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		res, err := resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
		})

		assert.NotNil(t, err, "should fail because email is not verified")
		assert.Nil(t, res)
		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		n, err := utils.EncryptNonce(verificationRequest.Nonce)
		assert.NoError(t, err)
		assert.NotEmpty(t, n)
		assert.NotNil(t, verificationRequest)
		res, err = resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		_, err = resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
			Roles:    []string{"test"},
		})
		assert.NotNil(t, err, "invalid roles")

		_, err = resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password + "s",
		})
		assert.NotNil(t, err, "invalid password")

		loginRes, err := resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
		})

		assert.Nil(t, err, "login successful")
		assert.NotNil(t, loginRes.AccessToken, "access token should not be empty")

		cleanData(email)
	})
}

package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/stretchr/testify/assert"
)

func loginTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should login`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "login." + s.TestInfo.Email
		signUpRes, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, signUpRes)
		res, err := resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    refs.NewStringRef(email),
			Password: s.TestInfo.Password,
		})
		// access token should be empty as email is not verified
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Nil(t, res.AccessToken)
		assert.NotEmpty(t, res.Message)
		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.NoError(t, err)
		assert.NotNil(t, verificationRequest)
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
			Email:    refs.NewStringRef(email),
			Password: s.TestInfo.Password,
			Roles:    []string{"test"},
		})
		assert.NotNil(t, err, "invalid roles")

		_, err = resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    refs.NewStringRef(email),
			Password: s.TestInfo.Password + "s",
		})
		assert.NotNil(t, err, "invalid password")

		loginRes, err := resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    refs.NewStringRef(email),
			Password: s.TestInfo.Password,
		})

		assert.Nil(t, err, "login successful")
		assert.NotNil(t, loginRes.AccessToken, "access token should not be empty")

		cleanData(email)
	})
}

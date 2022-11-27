package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func signupTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should complete the signup and check duplicates`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "signup." + s.TestInfo.Email
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password + "s",
		})
		assert.NotNil(t, err, "invalid password")
		assert.Nil(t, res)

		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        "test",
			ConfirmPassword: "test",
		})
		assert.NotNil(t, err, "invalid password")
		assert.Nil(t, res)

		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, true)
		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NotNil(t, err, "singup disabled")
		assert.Nil(t, res)

		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, false)
		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Nil(t, err, "signup should be successful")
		user := *res.User
		assert.Equal(t, email, user.Email)
		assert.Nil(t, res.AccessToken, "access token should be nil")

		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Nil(t, res)
		assert.NotNil(t, err, "should throw duplicate email error")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.Nil(t, err)
		assert.Equal(t, email, verificationRequest.Email)
		cleanData(email)
	})
}

package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func signupTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should complete the signup and check duplicates`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "signup." + s.TestInfo.Email
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password + "s",
		})
		assert.NotNil(t, err, "invalid password")
		assert.Nil(t, res)
		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        "test",
			ConfirmPassword: "test",
		})
		assert.NotNil(t, err, "invalid password")
		assert.Nil(t, res)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, true)
		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NotNil(t, err, "signup disabled")
		assert.Nil(t, res)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, false)
		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
			AppData: map[string]interface{}{
				"test": "test",
			},
		})
		assert.Nil(t, err, "signup should be successful")
		user := *res.User
		assert.Equal(t, email, refs.StringValue(user.Email))
		assert.Equal(t, "test", user.AppData["test"])
		assert.Nil(t, res.AccessToken, "access token should be nil")
		res, err = resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NotNil(t, err, "should throw duplicate email error")
		assert.Nil(t, res)
		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
		assert.Nil(t, err)
		assert.Equal(t, email, verificationRequest.Email)
		cleanData(email)
	})
}

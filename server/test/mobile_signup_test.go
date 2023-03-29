package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func mobileSingupTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should complete the signup with mobile and check duplicates`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "mobile_basic_auth_signup." + s.TestInfo.Email
		res, err := resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password + "s",
		})
		assert.NotNil(t, err, "invalid password")
		assert.Nil(t, res)

		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        "test",
			ConfirmPassword: "test",
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, true)
		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, false)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication, true)
		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableMobileBasicAuthentication, false)

		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			PhoneNumber:     "   ",
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			PhoneNumber:     "test",
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			PhoneNumber:     "1234567890",
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, res.AccessToken)
		assert.Equal(t, "1234567890@authorizer.dev", res.User.Email)

		res, err = resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			PhoneNumber:     "1234567890",
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		cleanData(email)
		cleanData("1234567890@authorizer.dev")
	})
}

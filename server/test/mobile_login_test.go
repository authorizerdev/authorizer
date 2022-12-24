package test

import (
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func mobileLoginTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should login via mobile`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "mobile_login." + s.TestInfo.Email
		phoneNumber := "2234567890"
		signUpRes, err := resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			Email:           refs.NewStringRef(email),
			PhoneNumber:     phoneNumber,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, signUpRes)
		assert.Equal(t, email, signUpRes.User.Email)
		assert.Equal(t, phoneNumber, refs.StringValue(signUpRes.User.PhoneNumber))
		assert.True(t, strings.Contains(signUpRes.User.SignupMethods, constants.AuthRecipeMethodMobileBasicAuth))
		assert.Len(t, strings.Split(signUpRes.User.SignupMethods, ","), 1)

		res, err := resolvers.MobileLoginResolver(ctx, model.MobileLoginInput{
			PhoneNumber: phoneNumber,
			Password:    "random_test",
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		// Should fail for email login
		res, err = resolvers.LoginResolver(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
		})
		assert.Error(t, err)
		assert.Nil(t, res)

		res, err = resolvers.MobileLoginResolver(ctx, model.MobileLoginInput{
			PhoneNumber: phoneNumber,
			Password:    s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, res.AccessToken)
		assert.NotEmpty(t, res.IDToken)

		cleanData(email)
	})
}

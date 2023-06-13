package test

import (
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func verifyMobileTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should verify mobile`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "mobile_verification." + s.TestInfo.Email
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

		// should fail because phone is not verified
		res, err = resolvers.MobileLoginResolver(ctx, model.MobileLoginInput{
			PhoneNumber: phoneNumber,
			Password:    s.TestInfo.Password,
		})
		assert.NotNil(t, err, "should fail because phone is not verified")
		assert.Nil(t, res)

		// get code from db
		smsRequest, err := db.Provider.GetCodeByPhone(ctx, phoneNumber)
		assert.NoError(t, err)
		assert.NotEmpty(t, smsRequest.Code)

		// throw an error if the code is not correct
		verifySMSRequest, err := resolvers.VerifyMobileResolver(ctx, model.VerifyMobileRequest{
			PhoneNumber: phoneNumber,
			Code:  "rand_12@1",
		})
		assert.NotNil(t, err, "should fail because of bad credentials")
		assert.Nil(t, verifySMSRequest)
	
		verifySMSRequest, err = resolvers.VerifyMobileResolver(ctx, model.VerifyMobileRequest{
			PhoneNumber: phoneNumber,
			Code:   smsRequest.Code,
		})
		assert.Nil(t, err)
		assert.NotEqual(t, verifySMSRequest.Message, "", "message should not be empty")
	
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

package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/db"
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
		// should fail because phone is not verified
		res, err = resolvers.MobileLoginResolver(ctx, model.MobileLoginInput{
			PhoneNumber: phoneNumber,
			Password:    s.TestInfo.Password,
		})
		assert.NotNil(t, err, "should fail because phone is not verified")
		assert.Nil(t, res)
		smsRequest, err := db.Provider.GetOTPByPhoneNumber(ctx, phoneNumber)
		assert.NoError(t, err)
		assert.NotEmpty(t, smsRequest.Otp)
		verifySMSRequest, err := resolvers.VerifyOtpResolver(ctx, model.VerifyOTPRequest{
			PhoneNumber: &phoneNumber,
			Otp:         smsRequest.Otp,
		})
		assert.Nil(t, err)
		assert.NotEqual(t, verifySMSRequest.Message, "", "message should not be empty")
		assert.NotEmpty(t, verifySMSRequest.AccessToken)
		assert.NotEmpty(t, verifySMSRequest.IDToken)
		cleanData(email)
	})
}

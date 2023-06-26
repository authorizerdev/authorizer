package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func forgotPasswordTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should run forgot password`, func(t *testing.T) {
		_, ctx := createContext(s)
		phoneNumber := "2234567890"
		email := "forgot_password." + s.TestInfo.Email
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		forgotPasswordRes, err := resolvers.ForgotPasswordResolver(ctx, model.ForgotPasswordInput{
			EmailOrPhone: email,
		})
		assert.Nil(t, err, "no errors for forgot password")
		assert.NotNil(t, forgotPasswordRes)
		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		assert.Nil(t, err)

		assert.Equal(t, verificationRequest.Identifier, constants.VerificationTypeForgotPassword)

		// Signup using phone and forget password
		signUpRes, err := resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			Email:           refs.NewStringRef(email),
			PhoneNumber:     phoneNumber,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, signUpRes)

		smsRequest, err := db.Provider.GetCodeByPhone(ctx, phoneNumber)
		assert.NoError(t, err)
		assert.NotEmpty(t, smsRequest.Code)

		verifySMSRequest, err := resolvers.VerifyMobileResolver(ctx, model.VerifyMobileRequest{
			PhoneNumber: phoneNumber,
			Code:   smsRequest.Code,
		})
		assert.Nil(t, err)
		assert.NotEqual(t, verifySMSRequest.Message, "", "message should not be empty")
	
		forgotPasswordWithPhone, err := resolvers.ForgotPasswordResolver(ctx, model.ForgotPasswordInput{
			EmailOrPhone: phoneNumber,
		})
		assert.Nil(t, err)
		assert.NotNil(t, forgotPasswordWithPhone, "verification code has been sent to your phone")

		cleanData(email)
	})
}

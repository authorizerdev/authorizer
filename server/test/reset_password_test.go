package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func resetPasswordTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should reset password`, func(t *testing.T) {
		phoneNumber := "2234567890"
		phonePointer := &phoneNumber
		email := "reset_password." + s.TestInfo.Email
		_, ctx := createContext(s)
		_, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		_, err = resolvers.ForgotPasswordResolver(ctx, model.ForgotPasswordInput{
			EmailOrPhone: email,
		})
		assert.Nil(t, err, "no errors for forgot password")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		assert.Nil(t, err, "should get forgot password request")
		assert.NotNil(t, verificationRequest)
		_, err = resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			TokenOrCode:           verificationRequest.Token,
			Password:        "test1",
			ConfirmPassword: "test",
		})

		assert.NotNil(t, err, "passowrds don't match")

		_, err = resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			TokenOrCode:           verificationRequest.Token,
			Password:        "test1",
			ConfirmPassword: "test1",
		})

		assert.NotNil(t, err, "invalid password")

		_, err = resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			TokenOrCode:           verificationRequest.Token,
			Password:              "Test@1234",
			ConfirmPassword:       "Test@1234",
		})

		assert.Nil(t, err, "password changed successfully")

		// Signup with phone, forget password and then - reset it.
		signUpRes, err := resolvers.MobileSignupResolver(ctx, &model.MobileSignUpInput{
			Email:           refs.NewStringRef(email),
			PhoneNumber:     phoneNumber,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, signUpRes)

		forgotPasswordWithPhone, err := resolvers.ForgotPasswordResolver(ctx, model.ForgotPasswordInput{
			EmailOrPhone: phoneNumber,
		})
		assert.Nil(t, err)
		assert.NotNil(t, forgotPasswordWithPhone)

		// get code from db
		smsRequestForReset, err := db.Provider.GetCodeByPhone(ctx, phoneNumber)
		assert.Nil(t, err)
		assert.NotNil(t, smsRequestForReset)

		// throw an error if the code is not correct
		resetPasswordResponse, err := resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			PhoneNumber: phonePointer,
			TokenOrCode:  "abcd@EFG",
		})
		assert.NotNil(t, err, "should fail because of bad credentials")
		assert.Nil(t, resetPasswordResponse)
	
		resetPasswordResponse, err = resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			PhoneNumber: phonePointer,
			TokenOrCode:   smsRequestForReset.Code,
		})
		assert.Nil(t, err)
		assert.NotNil(t, resetPasswordResponse)

		cleanData(email)
	})
}

package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func resetPasswordTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should reset password`, func(t *testing.T) {
		email := "reset_password." + s.TestInfo.Email
		_, ctx := createContext(s)
		_, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)

		_, err = resolvers.ForgotPasswordResolver(ctx, model.ForgotPasswordInput{
			Email: email,
		})
		assert.Nil(t, err, "no errors for forgot password")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		assert.Nil(t, err, "should get forgot password request")

		_, err = resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			Token:           verificationRequest.Token,
			Password:        "test1",
			ConfirmPassword: "test",
		})

		assert.NotNil(t, err, "passowrds don't match")

		_, err = resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			Token:           verificationRequest.Token,
			Password:        "test1",
			ConfirmPassword: "test1",
		})

		assert.NotNil(t, err, "invalid password")

		_, err = resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			Token:           verificationRequest.Token,
			Password:        "Test@1234",
			ConfirmPassword: "Test@1234",
		})

		assert.Nil(t, err, "password changed successfully")

		cleanData(email)
	})
}

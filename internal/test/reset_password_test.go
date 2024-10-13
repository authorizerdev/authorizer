package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func resetPasswordTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should reset password`, func(t *testing.T) {
		email := "reset_password." + s.TestInfo.Email
		_, ctx := createContext(s)
		_, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		_, err = resolvers.ForgotPasswordResolver(ctx, model.ForgotPasswordInput{
			Email: refs.NewStringRef(email),
		})
		assert.Nil(t, err, "no errors for forgot password")
		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		assert.Nil(t, err, "should get forgot password request")
		assert.NotNil(t, verificationRequest)
		_, err = resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			Token:           refs.NewStringRef(verificationRequest.Token),
			Password:        "test1",
			ConfirmPassword: "test",
		})
		assert.NotNil(t, err, "passwords don't match")
		_, err = resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			Token:           refs.NewStringRef(verificationRequest.Token),
			Password:        "test1",
			ConfirmPassword: "test1",
		})
		assert.NotNil(t, err, "invalid password")
		_, err = resolvers.ResetPasswordResolver(ctx, model.ResetPasswordInput{
			Token:           refs.NewStringRef(verificationRequest.Token),
			Password:        "Test@1234",
			ConfirmPassword: "Test@1234",
		})
		assert.Nil(t, err, "password changed successfully")
		cleanData(email)
	})
}

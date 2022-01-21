package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func forgotPasswordTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should run forgot password`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "forgot_password." + s.TestInfo.Email
		_, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err = resolvers.ForgotPasswordResolver(ctx, model.ForgotPasswordInput{
			Email: email,
		})
		assert.Nil(t, err, "no errors for forgot password")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(email, constants.VerificationTypeForgotPassword)
		assert.Nil(t, err)

		assert.Equal(t, verificationRequest.Identifier, constants.VerificationTypeForgotPassword)

		cleanData(email)
	})
}

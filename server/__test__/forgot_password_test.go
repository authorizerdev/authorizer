package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func forgotPasswordTest(s TestSetup, t *testing.T) {
	t.Run(`should run forgot password`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "forgot_password." + s.TestInfo.Email
		_, err := resolvers.Signup(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err = resolvers.ForgotPassword(ctx, model.ForgotPasswordInput{
			Email: email,
		})
		assert.Nil(t, err, "no errors for forgot password")

		verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.ForgotPassword.String())
		assert.Nil(t, err)

		assert.Equal(t, verificationRequest.Identifier, enum.ForgotPassword.String())

		cleanData(email)
	})
}

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

func forgotPasswordTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should run forgot password`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "forgot_password." + s.TestInfo.Email
		res, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		forgotPasswordRes, err := resolvers.ForgotPasswordResolver(ctx, model.ForgotPasswordInput{
			Email: refs.NewStringRef(email),
		})
		assert.Nil(t, err, "no errors for forgot password")
		assert.NotNil(t, forgotPasswordRes)
		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeForgotPassword)
		assert.Nil(t, err)

		assert.Equal(t, verificationRequest.Identifier, constants.VerificationTypeForgotPassword)

		cleanData(email)
	})
}

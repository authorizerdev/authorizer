package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func resendVerifyEmailTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should resend verification email`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "resend_verify_email." + s.TestInfo.Email
		_, err := resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		assert.NoError(t, err)
		_, err = resolvers.ResendVerifyEmailResolver(ctx, model.ResendVerifyEmailInput{
			Email:      email,
			Identifier: constants.VerificationTypeBasicAuthSignup,
		})
		assert.NoError(t, err)

		cleanData(email)
	})
}

package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func resendVerifyEmailTests(s TestSetup, t *testing.T) {
	email := "resend_verify_email." + s.TestInfo.Email
	_, err := resolvers.Signup(s.Ctx, model.SignUpInput{
		Email:           email,
		Password:        s.TestInfo.Password,
		ConfirmPassword: s.TestInfo.Password,
	})

	_, err = resolvers.ResendVerifyEmail(s.Ctx, model.ResendVerifyEmailInput{
		Email:      email,
		Identifier: enum.BasicAuthSignup.String(),
	})

	assert.Nil(t, err)

	cleanData(email)
}

package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func loginTests(s TestSetup, t *testing.T) {
	email := "login." + s.TestInfo.Email
	_, err := resolvers.Signup(s.Ctx, model.SignUpInput{
		Email:           email,
		Password:        s.TestInfo.Password,
		ConfirmPassword: s.TestInfo.Password,
	})

	_, err = resolvers.Login(s.Ctx, model.LoginInput{
		Email:    email,
		Password: s.TestInfo.Password,
	})

	assert.NotNil(t, err, "should fail because email is not verified")

	verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.BasicAuthSignup.String())
	resolvers.VerifyEmail(s.Ctx, model.VerifyEmailInput{
		Token: verificationRequest.Token,
	})

	_, err = resolvers.Login(s.Ctx, model.LoginInput{
		Email:    email,
		Password: s.TestInfo.Password,
		Roles:    []string{"test"},
	})
	assert.NotNil(t, err, "invalid roles")

	_, err = resolvers.Login(s.Ctx, model.LoginInput{
		Email:    email,
		Password: s.TestInfo.Password + "s",
	})
	assert.NotNil(t, err, "invalid password")

	loginRes, err := resolvers.Login(s.Ctx, model.LoginInput{
		Email:    email,
		Password: s.TestInfo.Password,
	})

	assert.Nil(t, err, "login successful")
	assert.Nil(t, loginRes.AccessToken, "access token should not be empty")

	cleanData(email)
}

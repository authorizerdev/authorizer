package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func signupTests(s TestSetup, t *testing.T) {
	email := "signup." + s.TestInfo.Email
	res, err := resolvers.Signup(s.Ctx, model.SignUpInput{
		Email:           email,
		Password:        s.TestInfo.Password,
		ConfirmPassword: s.TestInfo.Password + "s",
	})
	assert.NotNil(t, err, "invalid password errors")

	res, err = resolvers.Signup(s.Ctx, model.SignUpInput{
		Email:           email,
		Password:        s.TestInfo.Password,
		ConfirmPassword: s.TestInfo.Password,
	})

	user := *res.User
	assert.Equal(t, email, user.Email)
	assert.Nil(t, res.AccessToken, "access token should be nil")

	res, err = resolvers.Signup(s.Ctx, model.SignUpInput{
		Email:           email,
		Password:        s.TestInfo.Password,
		ConfirmPassword: s.TestInfo.Password,
	})

	assert.NotNil(t, err, "should throw duplicate email error")

	verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.BasicAuthSignup.String())
	assert.Nil(t, err)
	assert.Equal(t, email, verificationRequest.Email)
	cleanData(email)
}

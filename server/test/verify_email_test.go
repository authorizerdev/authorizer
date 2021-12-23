package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func verifyEmailTest(s TestSetup, t *testing.T) {
	email := "verify_email." + s.TestInfo.Email
	res, err := resolvers.Signup(s.Ctx, model.SignUpInput{
		Email:           email,
		Password:        s.TestInfo.Password,
		ConfirmPassword: s.TestInfo.Password,
	})

	user := *res.User
	assert.Equal(t, email, user.Email)
	assert.Nil(t, res.AccessToken, "access token should be nil")
	verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.BasicAuthSignup.String())
	assert.Nil(t, err)
	assert.Equal(t, email, verificationRequest.Email)

	verifyRes, err := resolvers.VerifyEmail(s.Ctx, model.VerifyEmailInput{
		Token: verificationRequest.Token,
	})
	assert.Nil(t, err)
	assert.NotEqual(t, verifyRes.AccessToken, "", "access token should not be empty")

	cleanData(email)
}

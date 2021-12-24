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
	t.Run(`should login`, func(t *testing.T) {
		_, ctx := createContext(s)
		email := "login." + s.TestInfo.Email
		_, err := resolvers.Signup(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err = resolvers.Login(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
		})

		assert.NotNil(t, err, "should fail because email is not verified")

		verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.BasicAuthSignup.String())
		resolvers.VerifyEmail(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		_, err = resolvers.Login(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
			Roles:    []string{"test"},
		})
		assert.NotNil(t, err, "invalid roles")

		_, err = resolvers.Login(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password + "s",
		})
		assert.NotNil(t, err, "invalid password")

		loginRes, err := resolvers.Login(ctx, model.LoginInput{
			Email:    email,
			Password: s.TestInfo.Password,
		})

		assert.Nil(t, err, "login successful")
		assert.NotNil(t, loginRes.AccessToken, "access token should not be empty")

		cleanData(email)
	})
}

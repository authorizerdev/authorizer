package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func sessionTests(s TestSetup, t *testing.T) {
	t.Run(`should allow access to profile with session only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "session." + s.TestInfo.Email

		resolvers.Signup(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err := resolvers.Session(ctx, []string{})
		assert.NotNil(t, err, "unauthorized")

		verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.BasicAuthSignup.String())
		verifyRes, err := resolvers.VerifyEmail(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		token := *verifyRes.AccessToken
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.EnvData.COOKIE_NAME, token))

		sessionRes, err := resolvers.Session(ctx, []string{})
		assert.Nil(t, err)

		newToken := *sessionRes.AccessToken
		assert.Equal(t, token, newToken, "tokens should be equal")

		cleanData(email)
	})
}

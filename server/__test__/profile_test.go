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

func profileTests(s TestSetup, t *testing.T) {
	t.Helper()
	t.Run(`should get profile only with token`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "profile." + s.TestInfo.Email

		resolvers.Signup(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err := resolvers.Profile(ctx)
		assert.NotNil(t, err, "unauthorized")

		verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.BasicAuthSignup.String())
		verifyRes, err := resolvers.VerifyEmail(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		token := *verifyRes.AccessToken
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.EnvData.COOKIE_NAME, token))
		profileRes, err := resolvers.Profile(ctx)
		assert.Nil(t, err)

		newEmail := *&profileRes.Email
		assert.Equal(t, email, newEmail, "emails should be equal")

		cleanData(email)
	})
}

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

func logoutTests(s TestSetup, t *testing.T) {
	t.Helper()
	t.Run(`should logout user`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "logout." + s.TestInfo.Email

		_, err := resolvers.MagicLinkLogin(ctx, model.MagicLinkLoginInput{
			Email: email,
		})

		verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.MagicLinkLogin.String())
		verifyRes, err := resolvers.VerifyEmail(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		token := *verifyRes.AccessToken
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.EnvData.COOKIE_NAME, token))
		_, err = resolvers.Logout(ctx)
		assert.Nil(t, err)
		_, err = resolvers.Profile(ctx)
		assert.NotNil(t, err, "unauthorized")
		cleanData(email)
	})
}

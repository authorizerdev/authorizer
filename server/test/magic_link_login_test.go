package test

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func magicLinkLoginTests(s TestSetup, t *testing.T) {
	t.Run(`should login with magic link`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "magic_link_login." + s.TestInfo.Email

		_, err := resolvers.MagicLinkLogin(ctx, model.MagicLinkLoginInput{
			Email: email,
		})
		assert.Nil(t, err)

		verificationRequest, err := db.Mgr.GetVerificationByEmail(email, enum.MagicLinkLogin.String())
		verifyRes, err := resolvers.VerifyEmail(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		token := *verifyRes.AccessToken
		req.Header.Add("Authorization", "Bearer "+token)
		_, err = resolvers.Profile(ctx)
		assert.Nil(t, err)

		cleanData(email)
	})
}

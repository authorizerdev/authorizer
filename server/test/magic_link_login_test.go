package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func magicLinkLoginTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should login with magic link`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "magic_link_login." + s.TestInfo.Email

		_, err := resolvers.MagicLinkLoginResolver(ctx, model.MagicLinkLoginInput{
			Email: email,
		})
		assert.Nil(t, err)

		verificationRequest, err := db.Mgr.GetVerificationByEmail(email, constants.VerificationTypeMagicLinkLogin)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		token := *verifyRes.AccessToken
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyCookieName).(string), token))
		_, err = resolvers.ProfileResolver(ctx)
		assert.Nil(t, err)

		cleanData(email)
	})
}

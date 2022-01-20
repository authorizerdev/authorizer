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

func logoutTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should logout user`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "logout." + s.TestInfo.Email

		_, err := resolvers.MagicLinkLoginResolver(ctx, model.MagicLinkLoginInput{
			Email: email,
		})

		verificationRequest, err := db.Mgr.GetVerificationByEmail(email, constants.VerificationTypeMagicLinkLogin)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		token := *verifyRes.AccessToken
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName), token))
		_, err = resolvers.LogoutResolver(ctx)
		assert.Nil(t, err)
		_, err = resolvers.ProfileResolver(ctx)
		assert.NotNil(t, err, "unauthorized")
		cleanData(email)
	})
}

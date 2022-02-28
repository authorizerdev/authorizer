package test

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/utils"
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

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(email, constants.VerificationTypeMagicLinkLogin)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		sessions := sessionstore.GetUserSessions(verifyRes.User.ID)
		fingerPrint := ""
		refreshToken := ""
		for key, val := range sessions {
			fingerPrint = key
			refreshToken = val
		}

		fingerPrintHash, _ := utils.EncryptAES([]byte(fingerPrint))

		token := *verifyRes.AccessToken
		cookie := fmt.Sprintf("%s=%s;%s=%s;%s=%s", envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".fingerprint", url.QueryEscape(string(fingerPrintHash)), envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".refresh_token", refreshToken, envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".access_token", token)

		req.Header.Set("Cookie", cookie)
		_, err = resolvers.LogoutResolver(ctx)
		assert.Nil(t, err)
		cleanData(email)
	})
}

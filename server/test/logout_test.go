package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/stretchr/testify/assert"
)

func logoutTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should logout user`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "logout." + s.TestInfo.Email

		magicLoginRes, err := resolvers.MagicLinkLoginResolver(ctx, model.MagicLinkLoginInput{
			Email: email,
		})
		assert.NoError(t, err)
		assert.NotNil(t, magicLoginRes)
		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeMagicLinkLogin)
		assert.NoError(t, err)
		assert.NotNil(t, verificationRequest)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})
		assert.NoError(t, err)
		assert.NotNil(t, verifyRes)
		accessToken := *verifyRes.AccessToken
		assert.NotEmpty(t, accessToken)
		claims, err := token.ParseJWTToken(accessToken)
		assert.NoError(t, err)
		assert.NotEmpty(t, claims)
		loginMethod := claims["login_method"]
		sessionKey := verifyRes.User.ID
		if loginMethod != nil && loginMethod != "" {
			sessionKey = loginMethod.(string) + ":" + verifyRes.User.ID
		}

		sessionToken, err := memorystore.Provider.GetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+claims["nonce"].(string))
		assert.NoError(t, err)
		assert.NotEmpty(t, sessionToken)

		cookie := fmt.Sprintf("%s=%s;", constants.AppCookieName+"_session", sessionToken)
		cookie = strings.TrimSuffix(cookie, ";")

		req.Header.Set("Cookie", cookie)
		_, err = resolvers.LogoutResolver(ctx)
		assert.Nil(t, err)
		cleanData(email)
	})
}

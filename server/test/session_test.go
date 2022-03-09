package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/stretchr/testify/assert"
)

func sessionTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should allow access to profile with session only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "session." + s.TestInfo.Email

		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           email,
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err := resolvers.SessionResolver(ctx, &model.SessionQueryInput{})
		assert.NotNil(t, err, "unauthorized")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(email, constants.VerificationTypeBasicAuthSignup)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})

		sessions := sessionstore.GetUserSessions(verifyRes.User.ID)
		cookie := ""
		token := *verifyRes.AccessToken
		// set all they keys in cookie one of them should be session cookie
		for key := range sessions {
			if key != token {
				cookie += fmt.Sprintf("%s=%s;", envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+"_session", key)
			}
		}
		cookie = strings.TrimSuffix(cookie, ";")

		req.Header.Set("Cookie", cookie)

		_, err = resolvers.SessionResolver(ctx, &model.SessionQueryInput{})
		assert.Nil(t, err)

		cleanData(email)
	})
}

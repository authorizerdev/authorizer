package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/stretchr/testify/assert"
)

func sessionTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should allow access to profile with session only`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "session." + s.TestInfo.Email

		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})

		_, err := resolvers.SessionResolver(ctx, &model.SessionQueryInput{})
		assert.NotNil(t, err, "unauthorized")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeBasicAuthSignup)
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

		sessionKey := constants.AuthRecipeMethodBasicAuth + ":" + verifyRes.User.ID
		sessionToken, err := memorystore.Provider.GetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+claims["nonce"].(string))
		assert.NoError(t, err)
		assert.NotEmpty(t, sessionToken)

		cookie := fmt.Sprintf("%s=%s;", constants.AppCookieName+"_session", sessionToken)
		cookie = strings.TrimSuffix(cookie, ";")

		req.Header.Set("Cookie", cookie)
		_, err = resolvers.SessionResolver(ctx, &model.SessionQueryInput{})
		assert.Nil(t, err)

		cleanData(email)
	})
}

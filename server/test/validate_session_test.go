package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/stretchr/testify/assert"
)

// ValidateSessionTests tests all the validate session resolvers
func validateSessionTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should validate session`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "validate_session." + s.TestInfo.Email

		resolvers.SignupResolver(ctx, model.SignUpInput{
			Email:           refs.NewStringRef(email),
			Password:        s.TestInfo.Password,
			ConfirmPassword: s.TestInfo.Password,
		})
		_, err := resolvers.ValidateSessionResolver(ctx, &model.ValidateSessionInput{})
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
		res, err := resolvers.ValidateSessionResolver(ctx, &model.ValidateSessionInput{
			Cookie: sessionToken,
		})
		assert.Nil(t, err)
		assert.True(t, res.IsValid)
		req.Header.Set("Cookie", cookie)
		res, err = resolvers.ValidateSessionResolver(ctx, &model.ValidateSessionInput{})
		assert.Nil(t, err)
		assert.True(t, res.IsValid)
		assert.Equal(t, res.User.ID, verifyRes.User.ID)
		cleanData(email)
	})
}

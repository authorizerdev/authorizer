package test

import (
	"context"
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
		envstore.EnvStoreObj.UpdateEnvVariable(constants.BoolStoreIdentifier, constants.EnvKeyDisableSignUp, true)
		_, err := resolvers.MagicLinkLoginResolver(ctx, model.MagicLinkLoginInput{
			Email: email,
		})
		assert.NotNil(t, err, "signup disabled")

		envstore.EnvStoreObj.UpdateEnvVariable(constants.BoolStoreIdentifier, constants.EnvKeyDisableSignUp, false)
		_, err = resolvers.MagicLinkLoginResolver(ctx, model.MagicLinkLoginInput{
			Email: email,
		})
		assert.Nil(t, err, "signup should be successful")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(email, constants.VerificationTypeMagicLinkLogin)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})
		assert.NoError(t, err)
		assert.NotNil(t, verifyRes.AccessToken)
		s.GinContext.Request.Header.Set("Authorization", "Bearer "+*verifyRes.AccessToken)
		ctx = context.WithValue(req.Context(), "GinContextKey", s.GinContext)
		_, err = resolvers.ProfileResolver(ctx)
		assert.Nil(t, err)
		s.GinContext.Request.Header.Set("Authorization", "")
		cleanData(email)
	})
}

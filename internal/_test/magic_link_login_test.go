package test

import (
	"context"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func magicLinkLoginTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should login with magic link`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "magic_link_login." + s.TestInfo.Email
		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, true)
		_, err := resolvers.MagicLinkLoginResolver(ctx, model.MagicLinkLoginInput{
			Email: email,
		})
		assert.NotNil(t, err, "signup disabled")

		memorystore.Provider.UpdateEnvVariable(constants.EnvKeyDisableSignUp, false)
		_, err = resolvers.MagicLinkLoginResolver(ctx, model.MagicLinkLoginInput{
			Email: email,
		})
		assert.Nil(t, err, "signup should be successful")

		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeMagicLinkLogin)
		assert.NoError(t, err)
		assert.NotNil(t, verificationRequest)
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

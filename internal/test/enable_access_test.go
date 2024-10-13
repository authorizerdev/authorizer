package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memorystore"
	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func enableAccessTest(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should enable access`, func(t *testing.T) {
		req, ctx := createContext(s)
		email := "enable_access." + s.TestInfo.Email
		_, err := resolvers.MagicLinkLoginResolver(ctx, model.MagicLinkLoginInput{
			Email: email,
		})
		assert.NoError(t, err)
		verificationRequest, err := db.Provider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeMagicLinkLogin)
		assert.NoError(t, err)
		assert.NotNil(t, verificationRequest)
		verifyRes, err := resolvers.VerifyEmailResolver(ctx, model.VerifyEmailInput{
			Token: verificationRequest.Token,
		})
		assert.NoError(t, err)
		assert.NotNil(t, verifyRes.AccessToken)
		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		res, err := resolvers.RevokeAccessResolver(ctx, model.UpdateAccessInput{
			UserID: verifyRes.User.ID,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, res.Message)

		res, err = resolvers.EnableAccessResolver(ctx, model.UpdateAccessInput{
			UserID: verifyRes.User.ID,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, res.Message)

		// it should allow login with enabled access
		res, err = resolvers.MagicLinkLoginResolver(ctx, model.MagicLinkLoginInput{
			Email: email,
		})
		assert.Nil(t, err)
		assert.NotEmpty(t, res.Message)

		cleanData(email)
	})
}

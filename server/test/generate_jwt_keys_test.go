package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func generateJWTkeyTest(t *testing.T, s TestSetup) {
	t.Helper()
	req, ctx := createContext(s)
	t.Run(`generate_jwt_keys`, func(t *testing.T) {
		t.Run(`should throw unauthorized`, func(t *testing.T) {
			res, err := resolvers.GenerateJWTKeysResolver(ctx, model.GenerateJWTKeysInput{
				Type: "HS256",
			})
			assert.Error(t, err)
			assert.Nil(t, res)
		})
		t.Run(`should throw invalid`, func(t *testing.T) {
			res, err := resolvers.GenerateJWTKeysResolver(ctx, model.GenerateJWTKeysInput{
				Type: "test",
			})
			assert.Error(t, err)
			assert.Nil(t, res)
		})

		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)

		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		t.Run(`should generate HS256 secret`, func(t *testing.T) {
			res, err := resolvers.GenerateJWTKeysResolver(ctx, model.GenerateJWTKeysInput{
				Type: "HS256",
			})
			assert.NoError(t, err)
			assert.NotEmpty(t, res.Secret)
		})

		t.Run(`should generate RS256 secret`, func(t *testing.T) {
			res, err := resolvers.GenerateJWTKeysResolver(ctx, model.GenerateJWTKeysInput{
				Type: "RS256",
			})
			assert.NoError(t, err)
			assert.NotEmpty(t, res.PrivateKey)
			assert.NotEmpty(t, res.PublicKey)
		})

		t.Run(`should generate ES256 secret`, func(t *testing.T) {
			res, err := resolvers.GenerateJWTKeysResolver(ctx, model.GenerateJWTKeysInput{
				Type: "ES256",
			})
			assert.NoError(t, err)
			assert.NotEmpty(t, res.PrivateKey)
			assert.NotEmpty(t, res.PublicKey)
		})
	})
}

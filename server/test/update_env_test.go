package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/crypto"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/stretchr/testify/assert"
)

func updateEnvTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should update envs`, func(t *testing.T) {
		req, ctx := createContext(s)
		originalAppURL := envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAppURL)

		data := model.UpdateEnvInput{}
		_, err := resolvers.UpdateEnvResolver(ctx, data)

		assert.NotNil(t, err)

		h, err := crypto.EncryptPassword(envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminCookieName), h))
		newURL := "https://test.com"
		disableLoginPage := true
		allowedOrigins := []string{"http://localhost:8080"}
		data = model.UpdateEnvInput{
			AppURL:           &newURL,
			DisableLoginPage: &disableLoginPage,
			AllowedOrigins:   allowedOrigins,
		}
		_, err = resolvers.UpdateEnvResolver(ctx, data)

		assert.Nil(t, err)
		assert.Equal(t, envstore.EnvStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAppURL), newURL)
		assert.True(t, envstore.EnvStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableLoginPage))
		assert.Equal(t, envstore.EnvStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyAllowedOrigins), allowedOrigins)

		disableLoginPage = false
		data = model.UpdateEnvInput{
			AppURL:           &originalAppURL,
			DisableLoginPage: &disableLoginPage,
			AllowedOrigins:   []string{"*"},
		}
		_, err = resolvers.UpdateEnvResolver(ctx, data)
		assert.Nil(t, err)
	})
}

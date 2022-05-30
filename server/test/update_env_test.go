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

func updateEnvTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should update envs`, func(t *testing.T) {
		req, ctx := createContext(s)
		originalAppURL := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAppURL)

		data := model.UpdateEnvInput{}
		_, err := resolvers.UpdateEnvResolver(ctx, data)

		assert.NotNil(t, err)

		h, err := crypto.EncryptPassword(memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminCookieName), h))
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
		assert.Equal(t, memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAppURL), newURL)
		assert.True(t, memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableLoginPage))
		assert.Equal(t, memorystore.Provider.GetSliceStoreEnvVariable(constants.EnvKeyAllowedOrigins), allowedOrigins)

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

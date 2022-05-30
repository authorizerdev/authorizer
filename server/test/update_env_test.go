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
		originalAppURL, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAppURL)
		assert.Nil(t, err)

		data := model.UpdateEnvInput{}
		_, err = resolvers.UpdateEnvResolver(ctx, data)

		assert.NotNil(t, err)

		adminSecret, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret)
		assert.Nil(t, err)
		h, err := crypto.EncryptPassword(adminSecret)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
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

		appURL, err := memorystore.Provider.GetStringStoreEnvVariable(constants.EnvKeyAppURL)
		assert.Nil(t, err)
		assert.Equal(t, appURL, newURL)

		isLoginPageDisabled, err := memorystore.Provider.GetBoolStoreEnvVariable(constants.EnvKeyDisableLoginPage)
		assert.Nil(t, err)
		assert.True(t, isLoginPageDisabled)

		storedOrigins, err := memorystore.Provider.GetSliceStoreEnvVariable(constants.EnvKeyAllowedOrigins)
		assert.Nil(t, err)
		assert.Equal(t, storedOrigins, allowedOrigins)

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

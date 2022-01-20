package test

import (
	"fmt"
	"log"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func updateEnvTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should update envs`, func(t *testing.T) {
		req, ctx := createContext(s)
		originalAppURL := envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAppURL)

		data := model.UpdateEnvInput{}
		_, err := resolvers.UpdateEnvResolver(ctx, data)
		log.Println("error:", err)
		assert.NotNil(t, err)

		h, err := utils.EncryptPassword(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminSecret))
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAdminCookieName), h))
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
		assert.Equal(t, envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyAppURL), newURL)
		assert.True(t, envstore.EnvInMemoryStoreObj.GetBoolStoreEnvVariable(constants.EnvKeyDisableLoginPage))
		assert.Equal(t, envstore.EnvInMemoryStoreObj.GetSliceStoreEnvVariable(constants.EnvKeyAllowedOrigins), allowedOrigins)

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

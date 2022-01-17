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

func updateConfigTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should update configs`, func(t *testing.T) {
		req, ctx := createContext(s)
		originalAppURL := envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAppURL).(string)

		data := model.UpdateConfigInput{}
		_, err := resolvers.UpdateConfigResolver(ctx, data)
		log.Println("error:", err)
		assert.NotNil(t, err)

		h, err := utils.EncryptPassword(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAdminSecret).(string))
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAdminCookieName).(string), h))
		newURL := "https://test.com"
		data = model.UpdateConfigInput{
			AppURL: &newURL,
		}
		_, err = resolvers.UpdateConfigResolver(ctx, data)
		log.Println("error:", err)
		assert.Nil(t, err)
		assert.Equal(t, envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAppURL).(string), newURL)
		data = model.UpdateConfigInput{
			AppURL: &originalAppURL,
		}
		_, err = resolvers.UpdateConfigResolver(ctx, data)
		assert.Nil(t, err)
	})
}

package test

import (
	"fmt"
	"log"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func updateConfigTests(s TestSetup, t *testing.T) {
	t.Helper()
	t.Run(`should update configs`, func(t *testing.T) {
		req, ctx := createContext(s)
		originalAppURL := constants.EnvData.APP_URL
		log.Println("=> originalAppURL:", constants.EnvData.APP_URL)

		data := model.UpdateConfigInput{}
		_, err := resolvers.UpdateConfigResolver(ctx, data)
		log.Println("error:", err)
		assert.NotNil(t, err)

		h, _ := utils.HashPassword(constants.EnvData.ADMIN_SECRET)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.EnvData.ADMIN_COOKIE_NAME, h))
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.EnvData.ADMIN_COOKIE_NAME, h))
		newURL := "https://test.com"
		data = model.UpdateConfigInput{
			AppURL: &newURL,
		}
		_, err = resolvers.UpdateConfigResolver(ctx, data)
		log.Println("error:", err)
		assert.Nil(t, err)
		assert.Equal(t, constants.EnvData.APP_URL, newURL)
		assert.NotEqual(t, constants.EnvData.APP_URL, originalAppURL)
		data = model.UpdateConfigInput{
			AppURL: &originalAppURL,
		}
		_, err = resolvers.UpdateConfigResolver(ctx, data)
		assert.Nil(t, err)
	})
}

package test

import (
	"fmt"
	"log"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func configTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get config`, func(t *testing.T) {
		req, ctx := createContext(s)
		_, err := resolvers.EnvResolver(ctx)
		log.Println("error:", err)
		assert.NotNil(t, err)

		h, err := utils.EncryptPassword(envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAdminSecret).(string))
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAdminCookieName).(string), h))
		res, err := resolvers.EnvResolver(ctx)

		assert.Nil(t, err)
		assert.Equal(t, *res.AdminSecret, envstore.EnvInMemoryStoreObj.GetEnvVariable(constants.EnvKeyAdminSecret).(string))
	})
}

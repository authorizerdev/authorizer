package test

import (
	"fmt"
	"log"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func configTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get config`, func(t *testing.T) {
		req, ctx := createContext(s)
		_, err := resolvers.ConfigResolver(ctx)
		log.Println("error:", err)
		assert.NotNil(t, err)

		h, err := utils.EncryptPassword(constants.EnvData.ADMIN_SECRET)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.EnvData.ADMIN_COOKIE_NAME, h))
		res, err := resolvers.ConfigResolver(ctx)

		assert.Nil(t, err)
		assert.Equal(t, *res.AdminSecret, constants.EnvData.ADMIN_SECRET)
	})
}

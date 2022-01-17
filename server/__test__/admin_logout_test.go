package test

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/resolvers"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/stretchr/testify/assert"
)

func adminLogoutTests(s TestSetup, t *testing.T) {
	t.Run(`should get admin session`, func(t *testing.T) {
		req, ctx := createContext(s)
		_, err := resolvers.AdminLogout(ctx)
		assert.NotNil(t, err)

		h, err := utils.HashPassword(constants.EnvData.ADMIN_SECRET)
		assert.Nil(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.EnvData.ADMIN_COOKIE_NAME, h))
		_, err = resolvers.AdminLogout(ctx)

		assert.Nil(t, err)
	})
}

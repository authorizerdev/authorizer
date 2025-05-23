package integration_tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
)

// TestAdminLogout tests the logout functionality of the Authorizer application admin.
func TestAdminLogout(t *testing.T) {
	// Initialize test setup
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	t.Run("should complete admin login", func(t *testing.T) {
		_, err := ts.GraphQLProvider.AdminLogout(ctx)
		require.NotNil(t, err)

		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		res, err := ts.GraphQLProvider.AdminLogout(ctx)
		require.Nil(t, err)
		assert.NotNil(t, res)
	})
}

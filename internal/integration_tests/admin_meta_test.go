package integration_tests

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdminMeta tests the _admin_meta query, the non-deprecated source of the
// configured roles for the dashboard. It must be super-admin gated and return
// the configured role lists.
func TestAdminMeta(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	t.Run("should fail without admin cookie", func(t *testing.T) {
		res, err := ts.GraphQLProvider.AdminMeta(ctx)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should return configured roles with a valid admin cookie", func(t *testing.T) {
		_, err := ts.GraphQLProvider.AdminLogin(ctx, &model.AdminLoginRequest{
			AdminSecret: cfg.AdminSecret,
		})
		require.NoError(t, err)

		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		res, err := ts.GraphQLProvider.AdminMeta(ctx)
		require.NoError(t, err)
		require.NotNil(t, res)
		// Mirrors the configured roles in getTestConfig.
		assert.Equal(t, cfg.Roles, res.Roles)
		assert.Equal(t, cfg.DefaultRoles, res.DefaultRoles)
		assert.Contains(t, res.Roles, "admin")
		// Non-null list contract: never nil, even when nothing is configured.
		assert.NotNil(t, res.ProtectedRoles)
	})
}

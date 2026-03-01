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

// TestAdminSession tests the _admin_session query
func TestAdminSession(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	t.Run("should fail without admin cookie", func(t *testing.T) {
		res, err := ts.GraphQLProvider.AdminSession(ctx)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should return session with valid admin cookie", func(t *testing.T) {
		// Admin login first
		adminLoginReq := &model.AdminLoginRequest{
			AdminSecret: cfg.AdminSecret,
		}
		_, err := ts.GraphQLProvider.AdminLogin(ctx, adminLoginReq)
		require.NoError(t, err)

		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		res, err := ts.GraphQLProvider.AdminSession(ctx)
		require.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.Message)
	})
}

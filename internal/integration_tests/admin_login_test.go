package integration_tests

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdminLogin tests the login functionality of the Authorizer application admin.
func TestAdminLogin(t *testing.T) {
	// Initialize test setup
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	t.Run("should fail login with invalid admin secret", func(t *testing.T) {
		adminLoginReq := &model.AdminLoginInput{
			AdminSecret: "invalid_secret",
		}
		adminLoginRes, err := ts.GraphQLProvider.AdminLogin(ctx, adminLoginReq)
		require.Error(t, err)
		assert.Nil(t, adminLoginRes)
	})

	t.Run("should complete admin login", func(t *testing.T) {
		adminLoginReq := &model.AdminLoginInput{
			AdminSecret: cfg.AdminSecret,
		}
		res, err := ts.GraphQLProvider.AdminLogin(ctx, adminLoginReq)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

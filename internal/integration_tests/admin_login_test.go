package integration_tests

import (
	"testing"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdminSecretHeaderAuth tests admin authentication via x-authorizer-admin-secret header
func TestAdminSecretHeaderAuth(t *testing.T) {
	t.Run("should accept valid admin secret via header", func(t *testing.T) {
		cfg := getTestConfig()
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		req.Header.Set("x-authorizer-admin-secret", cfg.AdminSecret)
		users, err := ts.GraphQLProvider.Users(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)
		assert.NotNil(t, users)
	})

	t.Run("should reject invalid admin secret via header", func(t *testing.T) {
		cfg := getTestConfig()
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		req.Header.Set("x-authorizer-admin-secret", "wrong-secret")
		users, err := ts.GraphQLProvider.Users(ctx, &model.PaginatedRequest{})
		assert.Error(t, err)
		assert.Nil(t, users)
	})

	t.Run("should reject admin header when DisableAdminHeaderAuth is true", func(t *testing.T) {
		cfg := getTestConfig()
		cfg.DisableAdminHeaderAuth = true
		ts := initTestSetup(t, cfg)
		req, ctx := createContext(ts)

		req.Header.Set("x-authorizer-admin-secret", cfg.AdminSecret)
		users, err := ts.GraphQLProvider.Users(ctx, &model.PaginatedRequest{})
		assert.Error(t, err)
		assert.Nil(t, users)
	})
}

// TestAdminLogin tests the login functionality of the Authorizer application admin.
func TestAdminLogin(t *testing.T) {
	// Initialize test setup
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	t.Run("should fail login with invalid admin secret", func(t *testing.T) {
		adminLoginReq := &model.AdminLoginRequest{
			AdminSecret: "invalid_secret",
		}
		adminLoginRes, err := ts.GraphQLProvider.AdminLogin(ctx, adminLoginReq)
		require.Error(t, err)
		assert.Nil(t, adminLoginRes)
	})

	t.Run("should complete admin login", func(t *testing.T) {
		adminLoginReq := &model.AdminLoginRequest{
			AdminSecret: cfg.AdminSecret,
		}
		res, err := ts.GraphQLProvider.AdminLogin(ctx, adminLoginReq)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

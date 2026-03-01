package integration_tests

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUsers tests the _users admin query
func TestUsers(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create test users
	for i := 0; i < 3; i++ {
		email := fmt.Sprintf("users_test_%s_%d@authorizer.dev", uuid.New().String(), i)
		password := "Password@123"
		signupReq := &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		}
		_, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		require.NoError(t, err)
	}

	t.Run("should fail without admin auth", func(t *testing.T) {
		req.Header.Set("Cookie", "")
		users, err := ts.GraphQLProvider.Users(ctx, &model.PaginatedRequest{})
		assert.Error(t, err)
		assert.Nil(t, users)
	})

	t.Run("should list users with admin auth", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		users, err := ts.GraphQLProvider.Users(ctx, &model.PaginatedRequest{})
		require.NoError(t, err)
		assert.NotNil(t, users)
		assert.GreaterOrEqual(t, len(users.Users), 3)
		assert.NotNil(t, users.Pagination)
	})

	t.Run("should support pagination", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		limit := int64(2)
		page := int64(1)
		users, err := ts.GraphQLProvider.Users(ctx, &model.PaginatedRequest{
			Pagination: &model.PaginationRequest{
				Limit: &limit,
				Page:  &page,
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, users)
		assert.LessOrEqual(t, int64(len(users.Users)), limit)
	})
}

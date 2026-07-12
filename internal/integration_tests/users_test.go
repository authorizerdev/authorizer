package integration_tests

import (
	"fmt"
	"strings"
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
		users, err := ts.GraphQLProvider.Users(ctx, &model.ListUsersRequest{})
		assert.Error(t, err)
		assert.Nil(t, users)
	})

	t.Run("should list users with admin auth", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		users, err := ts.GraphQLProvider.Users(ctx, &model.ListUsersRequest{})
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
		users, err := ts.GraphQLProvider.Users(ctx, &model.ListUsersRequest{
			Pagination: &model.PaginationRequest{
				Limit: &limit,
				Page:  &page,
			},
		})
		require.NoError(t, err)
		assert.NotNil(t, users)
		assert.LessOrEqual(t, int64(len(users.Users)), limit)
	})

	t.Run("should filter users by case-insensitive search query", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		require.NoError(t, err)
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

		// Create a user with a unique token so the search matches exactly one.
		token := "srch" + uuid.New().String()[:8]
		email := fmt.Sprintf("%s@authorizer.dev", token)
		password := "Password@123"
		_, err = ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		})
		require.NoError(t, err)

		// Search is case-insensitive: query the uppercased token.
		query := strings.ToUpper(token)
		users, err := ts.GraphQLProvider.Users(ctx, &model.ListUsersRequest{Query: &query})
		require.NoError(t, err)
		require.NotNil(t, users)
		assert.Equal(t, int64(1), users.Pagination.Total)
		require.Len(t, users.Users, 1)
		assert.Equal(t, email, *users.Users[0].Email)

		// Search also matches the user id: a prefix of the id returns the user.
		userID := users.Users[0].ID
		idQuery := userID[:13]
		byID, err := ts.GraphQLProvider.Users(ctx, &model.ListUsersRequest{Query: &idQuery})
		require.NoError(t, err)
		require.NotNil(t, byID)
		foundByID := false
		for _, u := range byID.Users {
			if u.ID == userID {
				foundByID = true
				break
			}
		}
		assert.True(t, foundByID, "search by id prefix must return the user")

		// A query matching nothing returns an empty list.
		noMatch := "no-such-user-" + uuid.New().String()
		empty, err := ts.GraphQLProvider.Users(ctx, &model.ListUsersRequest{Query: &noMatch})
		require.NoError(t, err)
		require.NotNil(t, empty)
		assert.Equal(t, int64(0), empty.Pagination.Total)
		assert.Len(t, empty.Users, 0)
	})
}

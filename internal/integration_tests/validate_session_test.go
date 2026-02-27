package integration_tests

import (
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateSession add test cases for validating session tokens
func TestValidateSession(t *testing.T) {
	// Initialize test setup
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Test setup - create a test user
	email := "validate_session_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// Profile tests
	t.Run("after login", func(t *testing.T) {
		loginReq := &model.LoginRequest{
			Email:    &email,
			Password: password,
		}
		loginRes, err := ts.GraphQLProvider.Login(ctx, loginReq)
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)

		// Verify response contains expected tokens
		assert.NotEmpty(t, loginRes.AccessToken)
		assert.NotNil(t, loginRes.User)
		assert.Equal(t, email, *loginRes.User.Email)
		assert.True(t, loginRes.User.EmailVerified)

		t.Run("should fail without cookie", func(t *testing.T) {
			res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{})
			assert.Error(t, err)
			assert.Nil(t, res)
		})

		t.Run("should fail with invalid cookie", func(t *testing.T) {
			res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
				Cookie: "invalid-token",
			})
			assert.Error(t, err)
			assert.Nil(t, res)
		})

		t.Run("should pass with valid input", func(t *testing.T) {
			allData, err := ts.MemoryStoreProvider.GetAllData()
			require.NoError(t, err)
			sessionToken := ""
			for k, v := range allData {
				if strings.Contains(k, constants.TokenTypeSessionToken) {
					sessionToken = v
					break
				}
			}
			res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
				Cookie: sessionToken,
			})
			require.NoError(t, err)
			require.NotNil(t, res)

			t.Run("should fail with invalid roles", func(t *testing.T) {
				res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
					Cookie: sessionToken,
					Roles:  []string{"invalid-role"},
				})
				assert.Error(t, err)
				assert.Nil(t, res)
			})
			t.Run("should pass with valid roles", func(t *testing.T) {
				res, err := ts.GraphQLProvider.ValidateSession(ctx, &model.ValidateSessionRequest{
					Cookie: sessionToken,
					Roles:  []string{"user"},
				})
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.True(t, res.IsValid)
			})
		})
	})
}

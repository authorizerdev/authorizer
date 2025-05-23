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

// TestValidateJWTToken add test cases for validating JWT tokens
func TestValidateJWTToken(t *testing.T) {
	// Initialize test setup
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Test setup - create a test user
	email := "validate_jwt_test_" + uuid.New().String() + "@authorizer.dev"
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

		t.Run("should fail without token", func(t *testing.T) {
			res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{})
			assert.Error(t, err)
			assert.Nil(t, res)
		})

		t.Run("should fail without token type", func(t *testing.T) {
			res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
				Token: "invalid-token",
			})
			assert.Error(t, err)
			assert.Nil(t, res)
		})

		t.Run("should fail with invalid token", func(t *testing.T) {
			res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
				Token:     "invalid-token",
				TokenType: constants.TokenTypeAccessToken,
			})
			assert.Error(t, err)
			assert.Nil(t, res)
		})

		t.Run("should pass with valid input", func(t *testing.T) {
			allData, err := ts.MemoryStoreProvider.GetAllData()
			require.NoError(t, err)
			accessToken := ""
			for k, v := range allData {
				if strings.Contains(k, constants.TokenTypeAccessToken) {
					accessToken = v
					break
				}
			}
			res, err := ts.GraphQLProvider.ValidateJWTToken(ctx, &model.ValidateJWTTokenRequest{
				Token:     accessToken,
				TokenType: constants.TokenTypeAccessToken,
			})
			require.NoError(t, err)
			require.NotNil(t, res)
		})
	})
}

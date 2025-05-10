package integration_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSession tests the session functionality
func TestSession(t *testing.T) {
	// Initialize test setup
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Test setup - create a test user
	email := "session_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	signupReq := &model.SignUpInput{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// Session tests
	t.Run("should login successfully with valid credentials", func(t *testing.T) {
		loginReq := &model.LoginInput{
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
			res, err := ts.GraphQLProvider.Session(ctx, &model.SessionQueryInput{})
			assert.Error(t, err)
			assert.Nil(t, res)
		})

		t.Run("should return new access token with cookie", func(t *testing.T) {
			allData, err := ts.MemoryStoreProvider.GetAllData()
			require.NoError(t, err)
			sessionToken := ""
			for k, v := range allData {
				if strings.Contains(k, constants.TokenTypeSessionToken) {
					sessionToken = v
					break
				}
			}
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AppCookieName+"_session", sessionToken))
			res, err := ts.GraphQLProvider.Session(ctx, &model.SessionQueryInput{})
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.NotEmpty(t, res.AccessToken)
			assert.NotEqual(t, res.AccessToken, res.RefreshToken)
			assert.Equal(t, email, *res.User.Email)
		})
	})
}

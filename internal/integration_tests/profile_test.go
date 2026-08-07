package integration_tests

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProfile tests the profile functionality
func TestProfile(t *testing.T) {
	// Initialize test setup
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Test setup - create a test user
	email := "profile_test_" + uuid.New().String() + "@authorizer.dev"
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

		t.Run("should fail without cookie and authorization header", func(t *testing.T) {
			res, err := ts.GraphQLProvider.Profile(ctx)
			assert.Error(t, err)
			assert.Nil(t, res)
		})

		t.Run("should return profile with browser session", func(t *testing.T) {
			sessionToken := latestAppSessionCookie(ts)
			require.NotEmpty(t, sessionToken)
			req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AppCookieName+"_session", sessionToken))
			defer func() {
				req.Header.Del("Cookie")
			}()
			res, err := ts.GraphQLProvider.Profile(ctx)
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, email, *res.Email)
		})

		t.Run("should return profile with authorization header", func(t *testing.T) {
			// Uses the token the API actually returned, not one scraped out of
			// the session store. The store holds a SHA-256 digest now precisely
			// so that reading it yields nothing presentable — scraping it back
			// out was only ever possible because the token sat there in clear.
			loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
				Email:    &email,
				Password: password,
			})
			require.NoError(t, err)
			require.NotNil(t, loginRes.AccessToken)
			accessToken := *loginRes.AccessToken
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
			defer func() {
				req.Header.Del("Authorization")
			}()
			res, err := ts.GraphQLProvider.Profile(ctx)
			require.NoError(t, err)
			require.NotNil(t, res)
			assert.Equal(t, email, *res.Email)
		})
	})
}

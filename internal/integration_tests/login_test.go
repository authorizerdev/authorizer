package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestLogin tests the login functionality of the Authorizer application.
func TestLogin(t *testing.T) {
	// Initialize test setup
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Test setup - create a test user
	email := "login_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// Login tests
	t.Run("should fail login with invalid email", func(t *testing.T) {
		invalidEmail := "invalid@email.com"
		loginReq := &model.LoginRequest{
			Email:    &invalidEmail,
			Password: password,
		}
		res, err := ts.GraphQLProvider.Login(ctx, loginReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail login with invalid password", func(t *testing.T) {
		loginReq := &model.LoginRequest{
			Email:    &email,
			Password: "WrongPassword@123",
		}
		res, err := ts.GraphQLProvider.Login(ctx, loginReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should login successfully with valid credentials", func(t *testing.T) {
		loginReq := &model.LoginRequest{
			Email:    &email,
			Password: password,
		}
		res, err := ts.GraphQLProvider.Login(ctx, loginReq)
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Verify response contains expected tokens
		assert.NotEmpty(t, res.AccessToken)
		assert.NotNil(t, res.User)
		assert.Equal(t, email, *res.User.Email)
		assert.True(t, res.User.EmailVerified)
	})

	t.Run("should fail login with empty credentials", func(t *testing.T) {
		loginReq := &model.LoginRequest{}
		res, err := ts.GraphQLProvider.Login(ctx, loginReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("mobile login", func(t *testing.T) {
		mobile := fmt.Sprintf("%d", time.Now().Add(10*time.Second).Unix())
		signUpReq := &model.SignUpRequest{
			PhoneNumber:     &mobile,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signUpReq)
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Login
		loginReq := &model.LoginRequest{
			PhoneNumber: &mobile,
			Password:    password,
		}
		res, err = ts.GraphQLProvider.Login(ctx, loginReq)
		assert.NoError(t, err)
		assert.NotEmpty(t, res.AccessToken)
		assert.NotNil(t, res.User)
		assert.Equal(t, mobile, *res.User.PhoneNumber)
		assert.True(t, res.User.PhoneNumberVerified)
	})
}

package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// loginTest is a test function that tests the login functionality
func loginTest(t *testing.T, s *testSetup) {
	t.Run("login test", func(t *testing.T) {
		_, ctx := createContext(s)

		// Test setup - create a test user
		email := "login_test_" + uuid.New().String() + "@authorizer.dev"
		password := "Password@123"

		signupReq := &model.SignUpInput{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := s.GraphQLProvider.SignUp(ctx, signupReq)
		assert.NoError(t, err)
		assert.NotNil(t, res)

		// Login tests
		t.Run("should fail login with invalid email", func(t *testing.T) {
			invalidEmail := "invalid@email.com"
			loginReq := &model.LoginInput{
				Email:    &invalidEmail,
				Password: password,
			}
			res, err := s.GraphQLProvider.Login(ctx, loginReq)
			assert.Error(t, err)
			assert.Nil(t, res)
		})

		t.Run("should fail login with invalid password", func(t *testing.T) {
			loginReq := &model.LoginInput{
				Email:    &email,
				Password: "WrongPassword@123",
			}
			res, err := s.GraphQLProvider.Login(ctx, loginReq)
			assert.Error(t, err)
			assert.Nil(t, res)
		})

		t.Run("should login successfully with valid credentials", func(t *testing.T) {
			loginReq := &model.LoginInput{
				Email:    &email,
				Password: password,
			}
			res, err := s.GraphQLProvider.Login(ctx, loginReq)
			assert.NoError(t, err)
			assert.NotNil(t, res)

			// Verify response contains expected tokens
			assert.NotEmpty(t, res.AccessToken)
			assert.NotNil(t, res.User)
			assert.Equal(t, email, *res.User.Email)
			assert.True(t, res.User.EmailVerified)
		})

		t.Run("should fail login with empty credentials", func(t *testing.T) {
			loginReq := &model.LoginInput{}
			res, err := s.GraphQLProvider.Login(ctx, loginReq)
			assert.Error(t, err)
			assert.Nil(t, res)
		})

		t.Run("mobile login", func(t *testing.T) {
			mobile := fmt.Sprintf("%d", time.Now().Add(10*time.Second).Unix())
			signUpReq := &model.SignUpInput{
				PhoneNumber:     &mobile,
				Password:        password,
				ConfirmPassword: password,
			}
			res, err := s.GraphQLProvider.SignUp(ctx, signUpReq)
			assert.NoError(t, err)
			assert.NotNil(t, res)

			// Login
			loginReq := &model.LoginInput{
				PhoneNumber: &mobile,
				Password:    password,
			}
			res, err = s.GraphQLProvider.Login(ctx, loginReq)
			assert.NoError(t, err)
			assert.NotEmpty(t, res.AccessToken)
			assert.NotNil(t, res.User)
			assert.Equal(t, mobile, *res.User.PhoneNumber)
			assert.True(t, res.User.PhoneNumberVerified)
		})
	})
}

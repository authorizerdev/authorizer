package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestSignup tests the signup functionality of the Authorizer application.
func TestSignup(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "signup_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	t.Run("should fail for missing email or phone number", func(t *testing.T) {
		signupReq := &model.SignUpInput{
			Password: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail for missing confirm password", func(t *testing.T) {
		signupReq := &model.SignUpInput{
			Email:    &email,
			Password: password,
		}

		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail for mismatch confirm password", func(t *testing.T) {
		signupReq := &model.SignUpInput{
			Email:           &email,
			Password:        password,
			ConfirmPassword: "test@123",
		}

		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail for weak password", func(t *testing.T) {
		signupReq := &model.SignUpInput{
			Email:           &email,
			Password:        "test",
			ConfirmPassword: "test",
		}

		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail for invalid email", func(t *testing.T) {
		invalidEmail := "test"
		signupReq := &model.SignUpInput{
			Email:           &invalidEmail,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should fail for invalid mobile number", func(t *testing.T) {
		invalidMobileNumber := "1243234"
		signupReq := &model.SignUpInput{
			PhoneNumber:     &invalidMobileNumber,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("should pass for valid email", func(t *testing.T) {
		signupReq := &model.SignUpInput{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotNil(t, res.User)

		t.Run("should fail for duplicate email", func(t *testing.T) {
			signupReq := &model.SignUpInput{
				Email:           &email,
				Password:        password,
				ConfirmPassword: password,
			}
			res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
			assert.Error(t, err)
			assert.Nil(t, res)
		})
	})

	t.Run("should pass for valid mobile number", func(t *testing.T) {
		mobileNumber := fmt.Sprintf("%d", time.Now().Unix())
		signupReq := &model.SignUpInput{
			PhoneNumber:     &mobileNumber,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		// Validate mobile number
		assert.Equal(t, mobileNumber, *res.User.PhoneNumber)
		assert.True(t, res.User.PhoneNumberVerified)
		// Auth formula should be basic auth based on mobile number
		assert.Contains(t, constants.AuthRecipeMethodMobileBasicAuth, res.User.SignupMethods)

		t.Run("should fail for duplicate mobile number", func(t *testing.T) {
			signupReq := &model.SignUpInput{
				PhoneNumber:     &mobileNumber,
				Password:        password,
				ConfirmPassword: password,
			}
			res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
			assert.Error(t, err)
			assert.Nil(t, res)
		})
	})
}

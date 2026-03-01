package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// TestUpdateProfilePasswordWithSingleAuthEnabled tests password change
// when only one of basic auth / mobile basic auth is enabled.
// Before the fix, this would fail because the OR condition blocked it.
func TestUpdateProfilePasswordWithSingleAuthEnabled(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableBasicAuthentication = true
	cfg.EnableMobileBasicAuthentication = false
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "update_profile_single_auth_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	_, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)

	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email:    &email,
		Password: password,
	})
	require.NoError(t, err)
	ts.GinContext.Request.Header.Set("Authorization", "Bearer "+*loginRes.AccessToken)

	t.Run("should allow password change with only basic auth enabled", func(t *testing.T) {
		newPassword := "NewPassword@123"
		updateReq := &model.UpdateProfileRequest{
			OldPassword:        refs.NewStringRef(password),
			NewPassword:        refs.NewStringRef(newPassword),
			ConfirmNewPassword: refs.NewStringRef(newPassword),
		}
		res, err := ts.GraphQLProvider.UpdateProfile(ctx, updateReq)
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})
}

// TestUpdateProfile tests the update profile functionality
// using the GraphQL API.
// It creates a user, updates the profile, and verifies
// the changes in the database.
func TestUpdateProfile(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user
	email := "update_profile_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	// Signup the user
	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	assert.NoError(t, err)
	assert.NotNil(t, signupRes)
	assert.Equal(t, email, *signupRes.User.Email)
	assert.NotEmpty(t, *signupRes.AccessToken)

	// Login to get fresh tokens
	loginReq := &model.LoginRequest{
		Email:    &email,
		Password: password,
	}
	loginRes, err := ts.GraphQLProvider.Login(ctx, loginReq)
	assert.NoError(t, err)
	assert.NotNil(t, loginRes)
	assert.NotEmpty(t, *loginRes.AccessToken)

	// Set the authorization header for authenticated requests
	ts.GinContext.Request.Header.Set("Authorization", "Bearer "+*loginRes.AccessToken)

	// Test cases
	t.Run("should fail update profile without authentication", func(t *testing.T) {
		// Clear authorization header
		ts.GinContext.Request.Header.Set("Authorization", "")
		defer func() {
			ts.GinContext.Request.Header.Set("Authorization", "Bearer "+*loginRes.AccessToken)
		}()

		updateReq := &model.UpdateProfileRequest{
			GivenName: refs.NewStringRef("Test"),
		}
		updateRes, err := ts.GraphQLProvider.UpdateProfile(ctx, updateReq)
		assert.Error(t, err)
		assert.Nil(t, updateRes)
	})

	t.Run("should update basic profile information", func(t *testing.T) {
		givenName := "John"
		familyName := "Doe"
		nickname := "Johnny"
		phoneNumber := "+1234567890"

		updateReq := &model.UpdateProfileRequest{
			GivenName:   refs.NewStringRef(givenName),
			FamilyName:  refs.NewStringRef(familyName),
			Nickname:    refs.NewStringRef(nickname),
			PhoneNumber: refs.NewStringRef(phoneNumber),
		}

		updateRes, err := ts.GraphQLProvider.UpdateProfile(ctx, updateReq)
		assert.NoError(t, err)
		assert.NotNil(t, updateRes)

		// Get the profile
		profile, err := ts.GraphQLProvider.Profile(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, givenName, *profile.GivenName)
		assert.Equal(t, familyName, *profile.FamilyName)
		assert.Equal(t, nickname, *profile.Nickname)
		assert.Equal(t, phoneNumber, *profile.PhoneNumber)
		assert.Equal(t, email, *profile.Email)
	})

	t.Run("should update profile picture", func(t *testing.T) {
		picture := "https://example.com/profile.jpg"

		updateReq := &model.UpdateProfileRequest{
			Picture: refs.NewStringRef(picture),
		}

		updateRes, err := ts.GraphQLProvider.UpdateProfile(ctx, updateReq)
		assert.NoError(t, err)
		assert.NotNil(t, updateRes)

		// Get the profile
		profile, err := ts.GraphQLProvider.Profile(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, picture, *profile.Picture)
	})

	t.Run("should update gender", func(t *testing.T) {
		gender := "male"

		updateReq := &model.UpdateProfileRequest{
			Gender: refs.NewStringRef(gender),
		}

		updateRes, err := ts.GraphQLProvider.UpdateProfile(ctx, updateReq)
		assert.NoError(t, err)
		assert.NotNil(t, updateRes)

		// Get the profile
		profile, err := ts.GraphQLProvider.Profile(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, profile)
	})

	t.Run("should update birthdate", func(t *testing.T) {
		birthdate := "1990-01-01"

		updateReq := &model.UpdateProfileRequest{
			Birthdate: refs.NewStringRef(birthdate),
		}

		updateRes, err := ts.GraphQLProvider.UpdateProfile(ctx, updateReq)
		assert.NoError(t, err)
		assert.NotNil(t, updateRes)

		// Get the profile
		profile, err := ts.GraphQLProvider.Profile(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, birthdate, *profile.Birthdate)
	})

	t.Run("should change password with valid old password", func(t *testing.T) {
		newPassword := "NewPassword@123"
		updateReq := &model.UpdateProfileRequest{
			OldPassword:        refs.NewStringRef(password),
			NewPassword:        refs.NewStringRef(newPassword),
			ConfirmNewPassword: refs.NewStringRef(newPassword),
		}
		updateRes, err := ts.GraphQLProvider.UpdateProfile(ctx, updateReq)
		assert.NoError(t, err)
		assert.NotNil(t, updateRes)

		// Verify new password works by logging in
		loginReq2 := &model.LoginRequest{
			Email:    &email,
			Password: newPassword,
		}
		loginRes2, err := ts.GraphQLProvider.Login(ctx, loginReq2)
		assert.NoError(t, err)
		assert.NotNil(t, loginRes2)

		// Reset password back and refresh token
		updateReq2 := &model.UpdateProfileRequest{
			OldPassword:        refs.NewStringRef(newPassword),
			NewPassword:        refs.NewStringRef(password),
			ConfirmNewPassword: refs.NewStringRef(password),
		}
		ts.GinContext.Request.Header.Set("Authorization", "Bearer "+*loginRes2.AccessToken)
		_, err = ts.GraphQLProvider.UpdateProfile(ctx, updateReq2)
		assert.NoError(t, err)

		// Restore original token
		loginRes, err = ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
			Email:    &email,
			Password: password,
		})
		assert.NoError(t, err)
		ts.GinContext.Request.Header.Set("Authorization", "Bearer "+*loginRes.AccessToken)
	})

	t.Run("should fail password change with wrong old password", func(t *testing.T) {
		newPassword := "NewPassword@123"
		updateReq := &model.UpdateProfileRequest{
			OldPassword:        refs.NewStringRef("WrongOldPassword@123"),
			NewPassword:        refs.NewStringRef(newPassword),
			ConfirmNewPassword: refs.NewStringRef(newPassword),
		}
		updateRes, err := ts.GraphQLProvider.UpdateProfile(ctx, updateReq)
		assert.Error(t, err)
		assert.Nil(t, updateRes)
	})

	t.Run("should fail password change without confirm password", func(t *testing.T) {
		updateReq := &model.UpdateProfileRequest{
			NewPassword: refs.NewStringRef("NewPassword@123"),
		}
		updateRes, err := ts.GraphQLProvider.UpdateProfile(ctx, updateReq)
		assert.Error(t, err)
		assert.Nil(t, updateRes)
	})

	t.Run("should update multiple fields at once", func(t *testing.T) {
		givenName := "Updated"
		familyName := "User"
		picture := "https://example.com/new-profile.jpg"

		updateReq := &model.UpdateProfileRequest{
			GivenName:  refs.NewStringRef(givenName),
			FamilyName: refs.NewStringRef(familyName),
			Picture:    refs.NewStringRef(picture),
		}

		updateRes, err := ts.GraphQLProvider.UpdateProfile(ctx, updateReq)
		assert.NoError(t, err)
		assert.NotNil(t, updateRes)

		// Get the profile
		profile, err := ts.GraphQLProvider.Profile(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, givenName, *profile.GivenName)
		assert.Equal(t, familyName, *profile.FamilyName)
		assert.Equal(t, picture, *profile.Picture)
	})
}

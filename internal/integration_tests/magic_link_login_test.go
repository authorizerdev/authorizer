package integration_tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestMagicLinkLogin tests the magic link login functionality of the Authorizer application.
func TestMagicLinkLogin(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "magic_link_user" + uuid.New().String() + "@authorizer.dev"

	t.Run("should fail for missing email", func(t *testing.T) {
		loginReq := &model.MagicLinkLoginRequest{}
		res, err := ts.GraphQLProvider.MagicLinkLogin(ctx, loginReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
	t.Run("should fail for invalid email", func(t *testing.T) {
		loginReq := &model.MagicLinkLoginRequest{
			Email: "invalid-email",
		}
		res, err := ts.GraphQLProvider.MagicLinkLogin(ctx, loginReq)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
	t.Run("should pass for valid email", func(t *testing.T) {
		res, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
			Email: email,
		})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.Message)

		verificationRequest, err := ts.StorageProvider.GetVerificationRequestByEmail(ctx, email, constants.VerificationTypeMagicLinkLogin)
		assert.NoError(t, err)
		assert.NotNil(t, verificationRequest)
		verifyRes, err := ts.GraphQLProvider.VerifyEmail(ctx, &model.VerifyEmailRequest{
			Token: verificationRequest.Token,
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, *verifyRes.AccessToken)

		// Set the Authorization header for the Profile request
		ts.GinContext.Request.Header.Set("Authorization", "Bearer "+*verifyRes.AccessToken)

		profile, err := ts.GraphQLProvider.Profile(ctx)
		assert.Nil(t, err)
		assert.NotNil(t, profile)

		// Clean up the header after the test
		ts.GinContext.Request.Header.Set("Authorization", "")
	})
}

package integration_tests

import (
	"fmt"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestDeleteEmailTemplate tests the delete email template functionality
func TestDeleteEmailTemplate(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "delete_email_template_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	// Signup the user
	signupReq := &model.SignUpInput{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)
	require.NotNil(t, signupRes)

	// Test without admin cookie first
	t.Run("should fail without admin cookie", func(t *testing.T) {
		deleteParams := &model.DeleteEmailTemplateRequest{
			ID: "some-id",
		}

		resp, err := ts.GraphQLProvider.DeleteEmailTemplate(ctx, deleteParams)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	// Add admin cookie for the rest of the tests
	h, err := crypto.EncryptPassword(cfg.AdminSecret)
	assert.Nil(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	t.Run("should fail with empty ID", func(t *testing.T) {
		deleteParams := &model.DeleteEmailTemplateRequest{
			ID: "",
		}

		resp, err := ts.GraphQLProvider.DeleteEmailTemplate(ctx, deleteParams)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "email template ID required")
	})

	t.Run("should fail with whitespace-only ID", func(t *testing.T) {
		deleteParams := &model.DeleteEmailTemplateRequest{
			ID: "   ",
		}

		resp, err := ts.GraphQLProvider.DeleteEmailTemplate(ctx, deleteParams)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "email template ID required")
	})

	t.Run("should fail for non-existent template", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		deleteParams := &model.DeleteEmailTemplateRequest{
			ID: nonExistentID,
		}

		resp, err := ts.GraphQLProvider.DeleteEmailTemplate(ctx, deleteParams)
		require.Error(t, err)
		require.Nil(t, resp)
		// The error message may vary depending on the storage implementation
		assert.NotEmpty(t, err.Error())
	})

	// First create a template to test deletion
	var templateID string
	t.Run("setup - create template for deletion", func(t *testing.T) {
		// Clean up any existing template
		existingTemplate, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeInviteMember)
		if err == nil && existingTemplate != nil {
			err = ts.StorageProvider.DeleteEmailTemplate(ctx, existingTemplate)
			require.NoError(t, err)
		}

		// Create a new template
		addParams := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeInviteMember,
			Subject:   "Invitation to Join",
			Template:  "You've been invited to join our platform: {{.URL}}",
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, addParams)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Get the created template to get its ID
		template, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeInviteMember)
		require.NoError(t, err)
		require.NotNil(t, template)
		templateID = template.ID
	})

	t.Run("should successfully delete template", func(t *testing.T) {
		deleteParams := &model.DeleteEmailTemplateRequest{
			ID: templateID,
		}

		resp, err := ts.GraphQLProvider.DeleteEmailTemplate(ctx, deleteParams)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Email templated deleted successfully")

		// Verify the template was deleted
		template, err := ts.StorageProvider.GetEmailTemplateByID(ctx, templateID)
		assert.Error(t, err) // Should get an error because template should be gone
		assert.Nil(t, template)
	})

	t.Run("should fail when trying to delete already deleted template", func(t *testing.T) {
		// Try to delete the same template again
		deleteParams := &model.DeleteEmailTemplateRequest{
			ID: templateID,
		}

		resp, err := ts.GraphQLProvider.DeleteEmailTemplate(ctx, deleteParams)
		require.Error(t, err)
		require.Nil(t, resp)
		// The error message may vary depending on the storage implementation
		assert.NotEmpty(t, err.Error())
	})

	// Test with multiple email templates
	var template1ID, template2ID string
	t.Run("setup - create multiple templates", func(t *testing.T) {
		// First, clean up any existing templates with these event names
		existingTemplate1, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeBasicAuthSignup)
		if err == nil && existingTemplate1 != nil {
			err = ts.StorageProvider.DeleteEmailTemplate(ctx, existingTemplate1)
			require.NoError(t, err)
		}

		existingTemplate2, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeForgotPassword)
		if err == nil && existingTemplate2 != nil {
			err = ts.StorageProvider.DeleteEmailTemplate(ctx, existingTemplate2)
			require.NoError(t, err)
		}

		// Create first template
		addParams1 := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeBasicAuthSignup,
			Subject:   "Verify Your Signup",
			Template:  "Please verify your signup: {{.URL}}",
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, addParams1)
		require.NoError(t, err)
		require.NotNil(t, resp)

		template1, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeBasicAuthSignup)
		require.NoError(t, err)
		require.NotNil(t, template1)
		template1ID = template1.ID

		// Create second template
		addParams2 := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeForgotPassword,
			Subject:   "Password Reset",
			Template:  "Please reset your password: {{.URL}}",
		}

		resp, err = ts.GraphQLProvider.AddEmailTemplate(ctx, addParams2)
		require.NoError(t, err)
		require.NotNil(t, resp)

		template2, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeForgotPassword)
		require.NoError(t, err)
		require.NotNil(t, template2)
		template2ID = template2.ID
	})

	t.Run("should delete first template without affecting second", func(t *testing.T) {
		deleteParams := &model.DeleteEmailTemplateRequest{
			ID: template1ID,
		}

		resp, err := ts.GraphQLProvider.DeleteEmailTemplate(ctx, deleteParams)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Email templated deleted successfully")

		// First template should be gone
		template1, err := ts.StorageProvider.GetEmailTemplateByID(ctx, template1ID)
		assert.Error(t, err)
		assert.Nil(t, template1)

		// Second template should still exist
		template2, err := ts.StorageProvider.GetEmailTemplateByID(ctx, template2ID)
		assert.NoError(t, err)
		assert.NotNil(t, template2)
		assert.Equal(t, constants.VerificationTypeForgotPassword, template2.EventName)
	})

	t.Run("should delete second template", func(t *testing.T) {
		deleteParams := &model.DeleteEmailTemplateRequest{
			ID: template2ID,
		}

		resp, err := ts.GraphQLProvider.DeleteEmailTemplate(ctx, deleteParams)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Email templated deleted successfully")

		// Second template should now be gone
		template2, err := ts.StorageProvider.GetEmailTemplateByID(ctx, template2ID)
		assert.Error(t, err)
		assert.Nil(t, template2)
	})
}

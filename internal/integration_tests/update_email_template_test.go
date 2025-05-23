package integration_tests

import (
	"fmt"
	"testing"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateEmailTemplate tests the update email template functionality by the admin
func TestUpdateEmailTemplate(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "update_email_template_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	// Signup the user
	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)
	require.NotNil(t, signupRes)

	// Test without admin cookie first
	t.Run("should fail without admin cookie", func(t *testing.T) {
		updateParams := &model.UpdateEmailTemplateRequest{
			ID:       "some-id",
			Subject:  refs.NewStringRef("Updated Subject"),
			Template: refs.NewStringRef("Updated Template"),
		}

		resp, err := ts.GraphQLProvider.UpdateEmailTemplate(ctx, updateParams)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	// Add admin cookie for the rest of the tests
	h, err := crypto.EncryptPassword(cfg.AdminSecret)
	assert.Nil(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	t.Run("should fail for non-existent template", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		updateParams := &model.UpdateEmailTemplateRequest{
			ID:       nonExistentID,
			Subject:  refs.NewStringRef("Updated Subject"),
			Template: refs.NewStringRef("Updated Template"),
		}

		resp, err := ts.GraphQLProvider.UpdateEmailTemplate(ctx, updateParams)
		require.Error(t, err)
		require.Nil(t, resp)
		// The error message may vary depending on the storage implementation
		assert.NotEmpty(t, err.Error())
	})

	// First create a template to test updating
	var templateID string
	t.Run("setup - create template for updating", func(t *testing.T) {
		// Clean up any existing template
		existingTemplate, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeForgotPassword)
		if err == nil && existingTemplate != nil {
			err = ts.StorageProvider.DeleteEmailTemplate(ctx, existingTemplate)
			require.NoError(t, err)
		}

		// Create a new template
		addParams := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeForgotPassword,
			Subject:   "Reset Your Password",
			Template:  "Click here to reset your password: {{.URL}}",
			Design:    refs.NewStringRef("original design"),
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, addParams)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Get the created template to get its ID
		template, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeForgotPassword)
		require.NoError(t, err)
		require.NotNil(t, template)
		templateID = template.ID
	})

	t.Run("should update subject", func(t *testing.T) {
		updateParams := &model.UpdateEmailTemplateRequest{
			ID:      templateID,
			Subject: refs.NewStringRef("Updated Reset Password Subject"),
		}

		resp, err := ts.GraphQLProvider.UpdateEmailTemplate(ctx, updateParams)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Email template updated successfully")

		// Verify the template was updated
		template, err := ts.StorageProvider.GetEmailTemplateByID(ctx, templateID)
		require.NoError(t, err)
		require.NotNil(t, template)
		assert.Equal(t, *updateParams.Subject, template.Subject)
		assert.Equal(t, constants.VerificationTypeForgotPassword, template.EventName) // Event name unchanged
	})

	t.Run("should update template content", func(t *testing.T) {
		updateParams := &model.UpdateEmailTemplateRequest{
			ID:       templateID,
			Template: refs.NewStringRef("New content for password reset: {{.URL}} - {{.AppName}}"),
		}

		resp, err := ts.GraphQLProvider.UpdateEmailTemplate(ctx, updateParams)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Email template updated successfully")

		// Verify the template was updated
		template, err := ts.StorageProvider.GetEmailTemplateByID(ctx, templateID)
		require.NoError(t, err)
		require.NotNil(t, template)
		assert.Equal(t, *updateParams.Template, template.Template)
	})

	t.Run("should update design", func(t *testing.T) {
		updateParams := &model.UpdateEmailTemplateRequest{
			ID:     templateID,
			Design: refs.NewStringRef("updated design with new styles"),
		}

		resp, err := ts.GraphQLProvider.UpdateEmailTemplate(ctx, updateParams)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Email template updated successfully")

		// Verify the template was updated
		template, err := ts.StorageProvider.GetEmailTemplateByID(ctx, templateID)
		require.NoError(t, err)
		require.NotNil(t, template)
		assert.Equal(t, *updateParams.Design, template.Design)
	})

	t.Run("should update event name", func(t *testing.T) {
		// First make sure the target event name doesn't exist yet
		existingTemplate, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeUpdateEmail)
		if err == nil && existingTemplate != nil {
			err = ts.StorageProvider.DeleteEmailTemplate(ctx, existingTemplate)
			require.NoError(t, err)
		}

		updateParams := &model.UpdateEmailTemplateRequest{
			ID:        templateID,
			EventName: refs.NewStringRef(constants.VerificationTypeUpdateEmail),
		}

		resp, err := ts.GraphQLProvider.UpdateEmailTemplate(ctx, updateParams)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Email template updated successfully")

		// Verify the template was updated
		template, err := ts.StorageProvider.GetEmailTemplateByID(ctx, templateID)
		require.NoError(t, err)
		require.NotNil(t, template)
		assert.Equal(t, constants.VerificationTypeUpdateEmail, template.EventName)
	})

	t.Run("should fail with invalid event name", func(t *testing.T) {
		updateParams := &model.UpdateEmailTemplateRequest{
			ID:        templateID,
			EventName: refs.NewStringRef("invalid_event_name"),
		}

		resp, err := ts.GraphQLProvider.UpdateEmailTemplate(ctx, updateParams)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid event name")
	})

	t.Run("should fail with whitespace-only subject", func(t *testing.T) {
		updateParams := &model.UpdateEmailTemplateRequest{
			ID:      templateID,
			Subject: refs.NewStringRef("   "),
		}

		resp, err := ts.GraphQLProvider.UpdateEmailTemplate(ctx, updateParams)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "empty subject not allowed")
	})

	t.Run("should fail with whitespace-only template", func(t *testing.T) {
		updateParams := &model.UpdateEmailTemplateRequest{
			ID:       templateID,
			Template: refs.NewStringRef("   "),
		}

		resp, err := ts.GraphQLProvider.UpdateEmailTemplate(ctx, updateParams)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "empty template not allowed")
	})

	t.Run("should fail with whitespace-only design", func(t *testing.T) {
		updateParams := &model.UpdateEmailTemplateRequest{
			ID:     templateID,
			Design: refs.NewStringRef("   "),
		}

		resp, err := ts.GraphQLProvider.UpdateEmailTemplate(ctx, updateParams)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "empty design not allowed")
	})

	t.Run("should update multiple fields at once", func(t *testing.T) {
		updateParams := &model.UpdateEmailTemplateRequest{
			ID:       templateID,
			Subject:  refs.NewStringRef("Multi-Update Subject"),
			Template: refs.NewStringRef("Multi-Update Template Content"),
			Design:   refs.NewStringRef("Multi-Update Design Content"),
		}

		resp, err := ts.GraphQLProvider.UpdateEmailTemplate(ctx, updateParams)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Email template updated successfully")

		// Verify the template was updated
		template, err := ts.StorageProvider.GetEmailTemplateByID(ctx, templateID)
		require.NoError(t, err)
		require.NotNil(t, template)
		assert.Equal(t, *updateParams.Subject, template.Subject)
		assert.Equal(t, *updateParams.Template, template.Template)
		assert.Equal(t, *updateParams.Design, template.Design)
	})
}

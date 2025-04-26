package integration_tests

import (
	"fmt"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestAddEmailTemplate tests the add email template functionality by the admin
func TestAddEmailTemplate(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "add_email_template_user_test_" + uuid.New().String() + "@authorizer.dev"
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
	require.NotNil(t, signupRes.User)

	t.Run("should fail without admin cookie", func(t *testing.T) {
		// Attempt to add template without admin authentication
		params := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeBasicAuthSignup,
			Subject:   "Verify your email",
			Template:  "Please verify your email by clicking the link: {{.URL}}",
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, params)
		require.Error(t, err)
		require.Nil(t, resp)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	// Add admin cookie for the rest of the tests
	h, err := crypto.EncryptPassword(cfg.AdminSecret)
	assert.Nil(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	t.Run("should fail with invalid event name", func(t *testing.T) {
		params := &model.AddEmailTemplateRequest{
			EventName: "invalid_event_name",
			Subject:   "Test Subject",
			Template:  "Test Template",
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, params)
		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("should fail with empty subject", func(t *testing.T) {
		params := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeBasicAuthSignup,
			Subject:   "",
			Template:  "Test Template",
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, params)
		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("should fail with whitespace-only subject", func(t *testing.T) {
		params := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeBasicAuthSignup,
			Subject:   "   ",
			Template:  "Test Template",
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, params)
		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("should fail with empty template", func(t *testing.T) {
		params := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeBasicAuthSignup,
			Subject:   "Test Subject",
			Template:  "",
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, params)
		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("should fail with whitespace-only template", func(t *testing.T) {
		params := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeBasicAuthSignup,
			Subject:   "Test Subject",
			Template:  "   ",
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, params)
		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("should add email template with design", func(t *testing.T) {
		// First clean up any existing template
		existingTemplate, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeMagicLinkLogin)
		if err == nil && existingTemplate != nil {
			err = ts.StorageProvider.DeleteEmailTemplate(ctx, existingTemplate)
			require.NoError(t, err)
		}

		params := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeMagicLinkLogin,
			Subject:   "Login with Magic Link",
			Template:  "Click this link to login: {{.URL}}",
			Design:    refs.NewStringRef("custom design"),
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Email template added successfully")

		// Verify template was saved
		template, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeMagicLinkLogin)
		require.NoError(t, err)
		require.NotNil(t, template)
		assert.Equal(t, params.EventName, template.EventName)
		assert.Equal(t, params.Subject, template.Subject)
		assert.Equal(t, params.Template, template.Template)
	})

	t.Run("should add email template without design", func(t *testing.T) {
		// First clean up any existing template
		existingTemplate, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeOTP)
		if err == nil && existingTemplate != nil {
			err = ts.StorageProvider.DeleteEmailTemplate(ctx, existingTemplate)
			require.NoError(t, err)
		}

		params := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeOTP,
			Subject:   "Your OTP Code",
			Template:  "Your OTP code is: {{.OTP}}",
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Email template added successfully")

		// Verify template was saved
		template, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeOTP)
		require.NoError(t, err)
		require.NotNil(t, template)
		assert.Equal(t, params.EventName, template.EventName)
		assert.Equal(t, params.Subject, template.Subject)
		assert.Equal(t, params.Template, template.Template)
		assert.Equal(t, "", template.Design)
	})

	t.Run("should add email template with empty design string", func(t *testing.T) {
		// First clean up any existing template
		existingTemplate, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeForgotPassword)
		if err == nil && existingTemplate != nil {
			err = ts.StorageProvider.DeleteEmailTemplate(ctx, existingTemplate)
			require.NoError(t, err)
		}

		params := &model.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeForgotPassword,
			Subject:   "Reset Your Password",
			Template:  "Click here to reset your password: {{.URL}}",
			Design:    refs.NewStringRef(""),
		}

		resp, err := ts.GraphQLProvider.AddEmailTemplate(ctx, params)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Contains(t, resp.Message, "Email template added successfully")

		// Verify template was saved
		template, err := ts.StorageProvider.GetEmailTemplateByEventName(ctx, constants.VerificationTypeForgotPassword)
		require.NoError(t, err)
		require.NotNil(t, template)
		assert.Equal(t, params.EventName, template.EventName)
		assert.Equal(t, params.Subject, template.Subject)
		assert.Equal(t, params.Template, template.Template)
		assert.Equal(t, "", template.Design)
	})
}

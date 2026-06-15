package integration_tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// seedEmailTemplate inserts a deterministic email template directly via storage
// and returns it. Used by the admin email template RPC tests that
// read/update/delete a template.
func seedEmailTemplate(t *testing.T, ts *testSetup, eventName string) *schemas.EmailTemplate {
	t.Helper()
	emailTemplate, err := ts.StorageProvider.AddEmailTemplate(context.Background(), &schemas.EmailTemplate{
		ID:        uuid.New().String(),
		EventName: eventName,
		Subject:   "seeded subject",
		Template:  "seeded template",
		Design:    "{}",
	})
	require.NoError(t, err)
	return emailTemplate
}

// TestAdminAddEmailTemplateGRPC exercises AuthorizerAdminService.AddEmailTemplate
// over gRPC: the fail-closed contract (no secret → Unauthenticated) and the
// happy path.
func TestAdminAddEmailTemplateGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.AddEmailTemplate(context.Background(), &authorizerv1.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeBasicAuthSignup,
			Subject:   "subject",
			Template:  "template",
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("invalid event name is an error", func(t *testing.T) {
		_, err := client.AddEmailTemplate(adminCtx(cfg.AdminSecret), &authorizerv1.AddEmailTemplateRequest{
			EventName: "not.a.real.event",
			Subject:   "subject",
			Template:  "template",
		})
		require.Error(t, err)
	})

	t.Run("adds email template with admin secret", func(t *testing.T) {
		resp, err := client.AddEmailTemplate(adminCtx(cfg.AdminSecret), &authorizerv1.AddEmailTemplateRequest{
			EventName: constants.VerificationTypeForgotPassword,
			Subject:   "Reset your password",
			Template:  "<p>reset</p>",
		})
		require.NoError(t, err)
		require.Equal(t, "Email template added successfully", resp.Message)
	})
}

// TestAdminUpdateEmailTemplateGRPC exercises
// AuthorizerAdminService.UpdateEmailTemplate over gRPC: the fail-closed contract
// and the happy path against a seeded template.
func TestAdminUpdateEmailTemplateGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	emailTemplate := seedEmailTemplate(t, ts, constants.VerificationTypeBasicAuthSignup)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		subject := "new subject"
		_, err := client.UpdateEmailTemplate(context.Background(), &authorizerv1.UpdateEmailTemplateRequest{
			Id:      emailTemplate.ID,
			Subject: &subject,
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("updates email template with admin secret", func(t *testing.T) {
		subject := "Updated subject"
		template := "<p>updated</p>"
		resp, err := client.UpdateEmailTemplate(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateEmailTemplateRequest{
			Id:       emailTemplate.ID,
			Subject:  &subject,
			Template: &template,
		})
		require.NoError(t, err)
		require.Equal(t, "Email template updated successfully.", resp.Message)
	})

	t.Run("updating unknown email template is an error", func(t *testing.T) {
		subject := "x"
		_, err := client.UpdateEmailTemplate(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateEmailTemplateRequest{
			Id:      uuid.New().String(),
			Subject: &subject,
		})
		require.Error(t, err)
	})
}

// TestAdminDeleteEmailTemplateGRPC exercises
// AuthorizerAdminService.DeleteEmailTemplate over gRPC: the fail-closed contract
// and the happy path against a seeded template.
func TestAdminDeleteEmailTemplateGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	emailTemplate := seedEmailTemplate(t, ts, constants.VerificationTypeBasicAuthSignup)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.DeleteEmailTemplate(context.Background(), &authorizerv1.DeleteEmailTemplateRequest{Id: emailTemplate.ID})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("deletes email template with admin secret", func(t *testing.T) {
		resp, err := client.DeleteEmailTemplate(adminCtx(cfg.AdminSecret), &authorizerv1.DeleteEmailTemplateRequest{Id: emailTemplate.ID})
		require.NoError(t, err)
		require.Equal(t, "Email templated deleted successfully", resp.Message)
	})

	t.Run("deleting unknown email template is an error", func(t *testing.T) {
		_, err := client.DeleteEmailTemplate(adminCtx(cfg.AdminSecret), &authorizerv1.DeleteEmailTemplateRequest{Id: uuid.New().String()})
		require.Error(t, err)
	})
}

// TestAdminEmailTemplatesGRPC exercises AuthorizerAdminService.EmailTemplates
// over gRPC: the fail-closed contract and the happy path with a seeded template
// present in the page.
func TestAdminEmailTemplatesGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	emailTemplate := seedEmailTemplate(t, ts, constants.VerificationTypeBasicAuthSignup)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.EmailTemplates(context.Background(), &authorizerv1.EmailTemplatesRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns email templates with admin secret", func(t *testing.T) {
		resp, err := client.EmailTemplates(adminCtx(cfg.AdminSecret), &authorizerv1.EmailTemplatesRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp.Pagination)
		found := false
		for _, et := range resp.EmailTemplates {
			if et.Id == emailTemplate.ID {
				found = true
				break
			}
		}
		require.True(t, found, "seeded email template should be present in the list")
	})
}

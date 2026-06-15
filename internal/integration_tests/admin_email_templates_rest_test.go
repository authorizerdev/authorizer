package integration_tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// TestAdminEmailTemplatesREST exercises the four admin email-template operations
// over the REST (grpc-gateway) surface. It mirrors
// admin_email_templates_grpc_test.go: every subtest asserts the fail-closed
// contract (no admin secret -> 401 unauthenticated) and the happy path with the
// x-authorizer-admin-secret header. REST JSON uses snake_case proto field names
// and wraps single objects/lists in the response message envelope.
func TestAdminEmailTemplatesREST(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	baseURL := newAdminRESTServer(t, ts)
	secret := cfg.AdminSecret

	t.Run("add_email_template", func(t *testing.T) {
		body := fmt.Sprintf(`{"event_name":%q,"subject":"Reset your password","template":"<p>reset</p>"}`,
			constants.VerificationTypeForgotPassword)

		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/add_email_template", "", body, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)

		var out struct {
			Message string `json:"message"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/add_email_template", secret, body, &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "Email template added successfully", out.Message)
	})

	t.Run("update_email_template", func(t *testing.T) {
		emailTemplate := seedEmailTemplate(t, ts, constants.VerificationTypeBasicAuthSignup)
		body := fmt.Sprintf(`{"id":%q,"subject":"Updated subject","template":"<p>updated</p>"}`,
			emailTemplate.ID)

		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/update_email_template", "", body, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)

		var out struct {
			Message string `json:"message"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/update_email_template", secret, body, &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "Email template updated successfully.", out.Message)
	})

	t.Run("delete_email_template", func(t *testing.T) {
		// Distinct event name per seeding subtest: email_template.event_name is
		// UNIQUE in storage (no timestamp suffix, unlike webhooks).
		emailTemplate := seedEmailTemplate(t, ts, constants.VerificationTypeMagicLinkLogin)
		body := fmt.Sprintf(`{"id":%q}`, emailTemplate.ID)

		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/delete_email_template", "", body, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)

		var out struct {
			Message string `json:"message"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/delete_email_template", secret, body, &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "Email templated deleted successfully", out.Message)
	})

	t.Run("email_templates", func(t *testing.T) {
		emailTemplate := seedEmailTemplate(t, ts, constants.VerificationTypeUpdateEmail)

		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/email_templates", "", `{}`, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)

		var out struct {
			EmailTemplates []struct {
				ID string `json:"id"`
			} `json:"email_templates"`
			Pagination map[string]any `json:"pagination"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/email_templates", secret, `{}`, &out)
		require.Equal(t, http.StatusOK, status)
		require.NotNil(t, out.Pagination)
		found := false
		for _, et := range out.EmailTemplates {
			if et.ID == emailTemplate.ID {
				found = true
				break
			}
		}
		require.True(t, found, "seeded email template should be present in the list")
	})
}

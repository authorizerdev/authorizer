package integration_tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestAdminWebhooksREST exercises the seven admin webhook operations over the
// REST (grpc-gateway) surface. It mirrors admin_webhooks_grpc_test.go: every
// subtest asserts the fail-closed contract (no admin secret -> 401
// unauthenticated) and the happy path with the x-authorizer-admin-secret header.
// REST JSON uses snake_case proto field names and wraps single objects/lists in
// the response message envelope (e.g. {"webhook":{...}}, {"webhooks":[...]}).
func TestAdminWebhooksREST(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	baseURL := newAdminRESTServer(t, ts)
	secret := cfg.AdminSecret

	t.Run("add_webhook", func(t *testing.T) {
		body := fmt.Sprintf(`{"event_name":%q,"endpoint":"https://example.com/hook-%s","enabled":true}`,
			constants.UserLoginWebhookEvent, uuid.New().String())

		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/add_webhook", "", body, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)

		var out struct {
			Message string `json:"message"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/add_webhook", secret, body, &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "Webhook added successfully", out.Message)
	})

	t.Run("update_webhook", func(t *testing.T) {
		// Each seeding subtest uses a distinct event name: the SQL provider
		// enforces a UNIQUE constraint on event_name and the appended
		// "-TIMESTAMP" suffix collides for same-second inserts of the same event.
		webhook := seedWebhook(t, ts, constants.UserCreatedWebhookEvent)
		body := fmt.Sprintf(`{"id":%q,"enabled":false,"endpoint":"https://example.com/updated-%s"}`,
			webhook.ID, uuid.New().String())

		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/update_webhook", "", body, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)

		var out struct {
			Message string `json:"message"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/update_webhook", secret, body, &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "Webhook updated successfully.", out.Message)
	})

	t.Run("delete_webhook", func(t *testing.T) {
		webhook := seedWebhook(t, ts, constants.UserSignUpWebhookEvent)
		body := fmt.Sprintf(`{"id":%q}`, webhook.ID)

		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/delete_webhook", "", body, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)

		var out struct {
			Message string `json:"message"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/delete_webhook", secret, body, &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "Webhook deleted successfully", out.Message)
	})

	t.Run("webhook", func(t *testing.T) {
		// The GetWebhook RPC is exposed at the REST path /v1/admin/webhook.
		webhook := seedWebhook(t, ts, constants.UserAccessRevokedWebhookEvent)
		body := fmt.Sprintf(`{"id":%q}`, webhook.ID)

		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/webhook", "", body, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)

		var out struct {
			Webhook struct {
				ID        string `json:"id"`
				EventName string `json:"event_name"`
			} `json:"webhook"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/webhook", secret, body, &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, webhook.ID, out.Webhook.ID)
		// Storage appends a "-TIMESTAMP" suffix to event names (legacy multi-hook
		// support), so assert the prefix rather than equality.
		require.Contains(t, out.Webhook.EventName, constants.UserAccessRevokedWebhookEvent)
	})

	t.Run("webhooks", func(t *testing.T) {
		webhook := seedWebhook(t, ts, constants.UserAccessEnabledWebhookEvent)

		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/webhooks", "", `{}`, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)

		var out struct {
			Webhooks []struct {
				ID string `json:"id"`
			} `json:"webhooks"`
			Pagination map[string]any `json:"pagination"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/webhooks", secret, `{}`, &out)
		require.Equal(t, http.StatusOK, status)
		require.NotNil(t, out.Pagination)
		found := false
		for _, w := range out.Webhooks {
			if w.ID == webhook.ID {
				found = true
				break
			}
		}
		require.True(t, found, "seeded webhook should be present in the list")
	})

	t.Run("webhook_logs", func(t *testing.T) {
		webhook := seedWebhook(t, ts, constants.UserDeletedWebhookEvent)
		_, err := ts.StorageProvider.AddWebhookLog(context.Background(), &schemas.WebhookLog{
			ID:         uuid.New().String(),
			HttpStatus: http.StatusOK,
			Response:   "ok",
			Request:    "{}",
			WebhookID:  webhook.ID,
		})
		require.NoError(t, err)

		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/webhook_logs", "", `{}`, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)

		body := fmt.Sprintf(`{"webhook_id":%q}`, webhook.ID)
		var out struct {
			WebhookLogs []struct {
				ID        string `json:"id"`
				WebhookID string `json:"webhook_id"`
			} `json:"webhook_logs"`
			Pagination map[string]any `json:"pagination"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/webhook_logs", secret, body, &out)
		require.Equal(t, http.StatusOK, status)
		require.NotNil(t, out.Pagination)
		require.NotEmpty(t, out.WebhookLogs)
		require.Equal(t, webhook.ID, out.WebhookLogs[0].WebhookID)
	})

	t.Run("test_endpoint", func(t *testing.T) {
		// TestEndpoint points at a local httptest.Server so no real network is
		// hit (getTestConfig sets SkipTestEndpointSSRFValidation=true).
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"received":true}`))
		}))
		t.Cleanup(srv.Close)

		body := fmt.Sprintf(`{"endpoint":%q,"event_name":%q}`, srv.URL, constants.UserLoginWebhookEvent)

		var env struct {
			Code string `json:"code"`
		}
		status := adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/test_endpoint", "", body, &env)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Equal(t, "unauthenticated", env.Code)

		// http_status is an int64 proto field, serialized as a JSON string.
		var out struct {
			HTTPStatus string `json:"http_status"`
			Response   string `json:"response"`
		}
		status = adminRESTJSON(t, baseURL, http.MethodPost, "/v1/admin/test_endpoint", secret, body, &out)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, fmt.Sprintf("%d", http.StatusAccepted), out.HTTPStatus)
		require.Contains(t, out.Response, "received")
	})
}

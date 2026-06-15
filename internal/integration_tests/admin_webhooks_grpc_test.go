package integration_tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// seedWebhook inserts a deterministic webhook directly via storage and returns
// it. Used by the admin webhook RPC tests that read/update/delete a webhook.
func seedWebhook(t *testing.T, ts *testSetup, eventName string) *schemas.Webhook {
	t.Helper()
	webhook, err := ts.StorageProvider.AddWebhook(context.Background(), &schemas.Webhook{
		ID:               uuid.New().String(),
		EventName:        eventName,
		EventDescription: "seeded webhook",
		EndPoint:         "https://example.com/" + uuid.New().String(),
		Enabled:          true,
		Headers:          `{"x-test":"1"}`,
	})
	require.NoError(t, err)
	return webhook
}

// TestAdminAddWebhookGRPC exercises AuthorizerAdminService.AddWebhook over gRPC:
// the fail-closed contract (no secret → Unauthenticated) and the happy path.
func TestAdminAddWebhookGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.AddWebhook(context.Background(), &authorizerv1.AddWebhookRequest{
			EventName: constants.UserLoginWebhookEvent,
			Endpoint:  "https://example.com/hook",
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("invalid event name is an error", func(t *testing.T) {
		_, err := client.AddWebhook(adminCtx(cfg.AdminSecret), &authorizerv1.AddWebhookRequest{
			EventName: "not.a.real.event",
			Endpoint:  "https://example.com/hook",
		})
		require.Error(t, err)
	})

	t.Run("adds webhook with admin secret", func(t *testing.T) {
		resp, err := client.AddWebhook(adminCtx(cfg.AdminSecret), &authorizerv1.AddWebhookRequest{
			EventName: constants.UserLoginWebhookEvent,
			Endpoint:  "https://example.com/hook-" + uuid.New().String(),
			Enabled:   true,
		})
		require.NoError(t, err)
		require.Equal(t, "Webhook added successfully", resp.Message)
	})
}

// TestAdminUpdateWebhookGRPC exercises AuthorizerAdminService.UpdateWebhook over
// gRPC: the fail-closed contract and the happy path against a seeded webhook.
func TestAdminUpdateWebhookGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	webhook := seedWebhook(t, ts, constants.UserLoginWebhookEvent)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		enabled := false
		_, err := client.UpdateWebhook(context.Background(), &authorizerv1.UpdateWebhookRequest{
			Id:      webhook.ID,
			Enabled: &enabled,
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("updates webhook with admin secret", func(t *testing.T) {
		enabled := false
		newEndpoint := "https://example.com/updated-" + uuid.New().String()
		resp, err := client.UpdateWebhook(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateWebhookRequest{
			Id:       webhook.ID,
			Enabled:  &enabled,
			Endpoint: &newEndpoint,
		})
		require.NoError(t, err)
		require.Equal(t, "Webhook updated successfully.", resp.Message)
	})

	t.Run("updating unknown webhook is an error", func(t *testing.T) {
		_, err := client.UpdateWebhook(adminCtx(cfg.AdminSecret), &authorizerv1.UpdateWebhookRequest{
			Id: uuid.New().String(),
		})
		require.Error(t, err)
	})
}

// TestAdminDeleteWebhookGRPC exercises AuthorizerAdminService.DeleteWebhook over
// gRPC: the fail-closed contract and the happy path against a seeded webhook.
func TestAdminDeleteWebhookGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	webhook := seedWebhook(t, ts, constants.UserLoginWebhookEvent)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.DeleteWebhook(context.Background(), &authorizerv1.DeleteWebhookRequest{Id: webhook.ID})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("deletes webhook with admin secret", func(t *testing.T) {
		resp, err := client.DeleteWebhook(adminCtx(cfg.AdminSecret), &authorizerv1.DeleteWebhookRequest{Id: webhook.ID})
		require.NoError(t, err)
		require.Equal(t, "Webhook deleted successfully", resp.Message)
	})

	t.Run("deleting unknown webhook is an error", func(t *testing.T) {
		_, err := client.DeleteWebhook(adminCtx(cfg.AdminSecret), &authorizerv1.DeleteWebhookRequest{Id: uuid.New().String()})
		require.Error(t, err)
	})
}

// TestAdminGetWebhookGRPC exercises AuthorizerAdminService.GetWebhook over gRPC:
// the fail-closed contract and the happy path against a seeded webhook.
func TestAdminGetWebhookGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	webhook := seedWebhook(t, ts, constants.UserLoginWebhookEvent)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.GetWebhook(context.Background(), &authorizerv1.GetWebhookRequest{Id: webhook.ID})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns webhook with admin secret", func(t *testing.T) {
		resp, err := client.GetWebhook(adminCtx(cfg.AdminSecret), &authorizerv1.GetWebhookRequest{Id: webhook.ID})
		require.NoError(t, err)
		require.NotNil(t, resp.Webhook)
		require.Equal(t, webhook.ID, resp.Webhook.Id)
		// Storage appends a "-TIMESTAMP" suffix to event names (legacy
		// multi-hook support), so assert the prefix rather than equality.
		require.Contains(t, resp.Webhook.EventName, constants.UserLoginWebhookEvent)
	})

	t.Run("unknown webhook is an error", func(t *testing.T) {
		_, err := client.GetWebhook(adminCtx(cfg.AdminSecret), &authorizerv1.GetWebhookRequest{Id: uuid.New().String()})
		require.Error(t, err)
	})
}

// TestAdminWebhooksGRPC exercises AuthorizerAdminService.Webhooks over gRPC: the
// fail-closed contract and the happy path with a seeded webhook in the page.
func TestAdminWebhooksGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	webhook := seedWebhook(t, ts, constants.UserLoginWebhookEvent)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.Webhooks(context.Background(), &authorizerv1.WebhooksRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns webhooks with admin secret", func(t *testing.T) {
		resp, err := client.Webhooks(adminCtx(cfg.AdminSecret), &authorizerv1.WebhooksRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp.Pagination)
		found := false
		for _, w := range resp.Webhooks {
			if w.Id == webhook.ID {
				found = true
				break
			}
		}
		require.True(t, found, "seeded webhook should be present in the list")
	})
}

// TestAdminWebhookLogsGRPC exercises AuthorizerAdminService.WebhookLogs over
// gRPC: the fail-closed contract and the happy path. A log is seeded against a
// webhook and asserted present when filtering by that webhook id.
func TestAdminWebhookLogsGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	webhook := seedWebhook(t, ts, constants.UserLoginWebhookEvent)

	_, err := ts.StorageProvider.AddWebhookLog(context.Background(), &schemas.WebhookLog{
		ID:         uuid.New().String(),
		HttpStatus: http.StatusOK,
		Response:   "ok",
		Request:    "{}",
		WebhookID:  webhook.ID,
	})
	require.NoError(t, err)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.WebhookLogs(context.Background(), &authorizerv1.WebhookLogsRequest{})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("returns webhook logs with admin secret", func(t *testing.T) {
		webhookID := webhook.ID
		resp, err := client.WebhookLogs(adminCtx(cfg.AdminSecret), &authorizerv1.WebhookLogsRequest{
			WebhookId: &webhookID,
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Pagination)
		require.NotEmpty(t, resp.WebhookLogs)
		require.Equal(t, webhook.ID, resp.WebhookLogs[0].WebhookId)
	})
}

// TestAdminTestEndpointGRPC exercises AuthorizerAdminService.TestEndpoint over
// gRPC: the fail-closed contract and the happy path against a local
// httptest.Server (getTestConfig sets SkipTestEndpointSSRFValidation so no real
// network is hit).
func TestAdminTestEndpointGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	t.Cleanup(srv.Close)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.TestEndpoint(context.Background(), &authorizerv1.TestEndpointRequest{
			Endpoint:  srv.URL,
			EventName: constants.UserLoginWebhookEvent,
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("invalid event name is an error", func(t *testing.T) {
		_, err := client.TestEndpoint(adminCtx(cfg.AdminSecret), &authorizerv1.TestEndpointRequest{
			Endpoint:  srv.URL,
			EventName: "not.a.real.event",
		})
		require.Error(t, err)
	})

	t.Run("calls endpoint and returns status with admin secret", func(t *testing.T) {
		resp, err := client.TestEndpoint(adminCtx(cfg.AdminSecret), &authorizerv1.TestEndpointRequest{
			Endpoint:  srv.URL,
			EventName: constants.UserLoginWebhookEvent,
		})
		require.NoError(t, err)
		require.Equal(t, int64(http.StatusAccepted), resp.HttpStatus)
		require.Contains(t, resp.Response, "received")
	})
}

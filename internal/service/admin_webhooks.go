package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// testEndpointHTTPTimeout bounds the outbound request TestEndpoint makes to a
// webhook endpoint so a slow/unresponsive target cannot stall the admin API.
const testEndpointHTTPTimeout = time.Second * 30

// AddWebhook registers a new webhook for an event and returns a status message.
// Requires super-admin auth. Logic migrated from internal/graphql/add_webhook.go.
func (p *provider) AddWebhook(ctx context.Context, meta RequestMetadata, params *model.AddWebhookRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "AddWebhook").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	if !validators.IsValidWebhookEventName(params.EventName) {
		log.Debug().Str("EventName", params.EventName).Msg("Invalid Event Name")
		return nil, nil, fmt.Errorf("invalid event name %s", params.EventName)
	}
	if strings.TrimSpace(params.Endpoint) == "" {
		log.Debug().Msg("endpoint is missing")
		return nil, nil, fmt.Errorf("empty endpoint not allowed")
	}
	// SSRF protection: validate endpoint URL and resolved IPs (skip in test env).
	if p.Env != constants.TestEnv {
		if err := validators.ValidateEndpointURL(params.Endpoint); err != nil {
			log.Debug().Err(err).Str("endpoint", params.Endpoint).Msg("endpoint URL rejected by SSRF filter")
			return nil, nil, fmt.Errorf("invalid endpoint: %s", err.Error())
		}
	}

	headerBytes, err := json.Marshal(params.Headers)
	if err != nil {
		return nil, nil, err
	}

	if params.EventDescription == nil {
		params.EventDescription = refs.NewStringRef(strings.Join(strings.Split(params.EventName, "."), " "))
	}

	webhook, err := p.StorageProvider.AddWebhook(ctx, &schemas.Webhook{
		EventDescription: refs.StringValue(params.EventDescription),
		EventName:        params.EventName,
		EndPoint:         params.Endpoint,
		Enabled:          params.Enabled,
		Headers:          string(headerBytes),
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to add webhook in db")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminWebhookCreatedEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeWebhook,
		ResourceID:   webhook.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: `Webhook added successfully`,
	}, nil, nil
}

// UpdateWebhook updates an existing webhook's event, endpoint, headers, or
// enabled state and returns a status message. Requires super-admin auth. Logic
// migrated from internal/graphql/update_webhook.go.
func (p *provider) UpdateWebhook(ctx context.Context, meta RequestMetadata, params *model.UpdateWebhookRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "UpdateWebhook").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	webhook, err := p.StorageProvider.GetWebhookByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetWebhookByID")
		return nil, nil, err
	}

	var headersMap map[string]interface{}
	if err := json.Unmarshal([]byte(webhook.Headers), &headersMap); err != nil {
		log.Debug().Err(err).Msg("error un-marshalling headers")
	}
	headersString := ""
	if headersMap != nil {
		headerBytes, err := json.Marshal(webhook.Headers)
		if err != nil {
			log.Debug().Err(err).Msg("failed to marshall headers")
			return nil, nil, err
		}
		headersString = string(headerBytes)
	}

	webhookDetails := &schemas.Webhook{
		ID:               webhook.ID,
		Key:              webhook.ID,
		EventName:        webhook.EventName,
		EventDescription: webhook.EventDescription,
		EndPoint:         webhook.EndPoint,
		Enabled:          webhook.Enabled,
		Headers:          headersString,
		CreatedAt:        webhook.CreatedAt,
	}

	if params.EventName != nil && webhookDetails.EventName != refs.StringValue(params.EventName) {
		if isValid := validators.IsValidWebhookEventName(refs.StringValue(params.EventName)); !isValid {
			log.Debug().Str("event_name", refs.StringValue(params.EventName)).Msg("invalid event name")
			return nil, nil, fmt.Errorf("invalid event name %s", refs.StringValue(params.EventName))
		}
		webhookDetails.EventName = refs.StringValue(params.EventName)
	}
	if params.Endpoint != nil && webhookDetails.EndPoint != refs.StringValue(params.Endpoint) {
		if strings.TrimSpace(refs.StringValue(params.Endpoint)) == "" {
			log.Debug().Msg("empty endpoint not allowed")
			return nil, nil, fmt.Errorf("empty endpoint not allowed")
		}
		// SSRF protection: validate endpoint URL and resolved IPs (skip in test env).
		if p.Env != constants.TestEnv {
			if err := validators.ValidateEndpointURL(refs.StringValue(params.Endpoint)); err != nil {
				log.Debug().Err(err).Str("endpoint", refs.StringValue(params.Endpoint)).Msg("endpoint URL rejected by SSRF filter")
				return nil, nil, fmt.Errorf("invalid endpoint: %s", err.Error())
			}
		}
		webhookDetails.EndPoint = refs.StringValue(params.Endpoint)
	}
	if params.Enabled != nil && webhookDetails.Enabled != refs.BoolValue(params.Enabled) {
		webhookDetails.Enabled = refs.BoolValue(params.Enabled)
	}
	if params.EventDescription != nil && webhookDetails.EventDescription != refs.StringValue(params.EventDescription) {
		webhookDetails.EventDescription = refs.StringValue(params.EventDescription)
	}
	if params.Headers != nil {
		headerBytes, err := json.Marshal(params.Headers)
		if err != nil {
			log.Debug().Err(err).Msg("failed to marshall headers")
			return nil, nil, err
		}
		webhookDetails.Headers = string(headerBytes)
	}

	if _, err := p.StorageProvider.UpdateWebhook(ctx, webhookDetails); err != nil {
		log.Debug().Err(err).Msg("failed UpdateWebhook")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminWebhookUpdatedEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeWebhook,
		ResourceID:   params.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: `Webhook updated successfully.`,
	}, nil, nil
}

// DeleteWebhook deletes a webhook by id and returns a status message. Requires
// super-admin auth. Logic migrated from internal/graphql/delete_webhook.go.
func (p *provider) DeleteWebhook(ctx context.Context, meta RequestMetadata, params *model.WebhookRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "DeleteWebhook").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	if params.ID == "" {
		log.Debug().Msg("Webhook ID required")
		return nil, nil, fmt.Errorf("webhook ID required")
	}

	log = log.With().Str("webhookID", params.ID).Logger()

	webhook, err := p.StorageProvider.GetWebhookByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get webhook by ID")
		return nil, nil, err
	}

	if err := p.StorageProvider.DeleteWebhook(ctx, webhook); err != nil {
		log.Debug().Err(err).Msg("Failed to delete webhook")
		return nil, nil, err
	}

	p.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminWebhookDeletedEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeWebhook,
		ResourceID:   params.ID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{
		Message: "Webhook deleted successfully",
	}, nil, nil
}

// Webhook returns a single webhook by id. Requires super-admin auth. Logic
// migrated from internal/graphql/webhook.go.
func (p *provider) Webhook(ctx context.Context, meta RequestMetadata, params *model.WebhookRequest) (*model.Webhook, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Webhook").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	webhook, err := p.StorageProvider.GetWebhookByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetWebhookByID")
		return nil, nil, err
	}
	return webhook.AsAPIWebhook(), nil, nil
}

// Webhooks returns a paginated list of webhooks. Requires super-admin auth.
// Logic migrated from internal/graphql/webhooks.go.
func (p *provider) Webhooks(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.Webhooks, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Webhooks").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	pagination := utils.GetPagination(params)
	webhooks, pagination, err := p.StorageProvider.ListWebhook(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListWebhook")
		return nil, nil, err
	}
	res := make([]*model.Webhook, len(webhooks))
	for i, webhook := range webhooks {
		res[i] = webhook.AsAPIWebhook()
	}
	return &model.Webhooks{
		Pagination: pagination,
		Webhooks:   res,
	}, nil, nil
}

// WebhookLogs returns a paginated list of webhook delivery logs, optionally
// filtered by webhook id. Requires super-admin auth. Logic migrated from
// internal/graphql/webhook_logs.go.
func (p *provider) WebhookLogs(ctx context.Context, meta RequestMetadata, params *model.ListWebhookLogRequest) (*model.WebhookLogs, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "WebhookLogs").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	var pagination *model.Pagination
	var webhookID string
	if params != nil {
		pagination = utils.GetPagination(&model.PaginatedRequest{
			Pagination: params.Pagination,
		})
		webhookID = refs.StringValue(params.WebhookID)
	} else {
		pagination = utils.GetPagination(nil)
		webhookID = ""
	}

	webhookLogs, pagination, err := p.StorageProvider.ListWebhookLogs(ctx, pagination, webhookID)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListWebhookLogs")
		return nil, nil, err
	}
	resItems := make([]*model.WebhookLog, len(webhookLogs))
	for i, webhookLog := range webhookLogs {
		resItems[i] = webhookLog.AsAPIWebhookLog()
	}
	return &model.WebhookLogs{
		Pagination:  pagination,
		WebhookLogs: resItems,
	}, nil, nil
}

// TestEndpoint sends a synthetic event payload to a webhook endpoint and returns
// the resulting HTTP status and response body. Requires super-admin auth. Logic
// migrated from internal/graphql/test_endpoint.go.
func (p *provider) TestEndpoint(ctx context.Context, meta RequestMetadata, params *model.TestEndpointRequest) (*model.TestEndpointResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "TestEndpoint").Logger()
	if err := p.requireSuperAdmin(meta); err != nil {
		return nil, nil, err
	}

	if !validators.IsValidWebhookEventName(params.EventName) {
		log.Debug().Str("event_name", params.EventName).Msg("Invalid event name")
		return nil, nil, fmt.Errorf("invalid event_name %s", params.EventName)
	}

	user := model.User{
		ID:            uuid.NewString(),
		Email:         refs.NewStringRef("test_endpoint@authorizer.dev"),
		EmailVerified: true,
		SignupMethods: constants.AuthRecipeMethodMagicLinkLogin,
		GivenName:     refs.NewStringRef("Foo"),
		FamilyName:    refs.NewStringRef("Bar"),
	}

	userBytes, err := json.Marshal(user)
	if err != nil {
		log.Debug().Err(err).Msg("error marshalling user obj")
		return nil, nil, err
	}
	userMap := map[string]interface{}{}
	if err := json.Unmarshal(userBytes, &userMap); err != nil {
		log.Debug().Err(err).Msg("error un-marshalling user obj")
		return nil, nil, err
	}

	reqBody := map[string]interface{}{
		"event_name": constants.UserLoginWebhookEvent,
		"user":       userMap,
	}
	if params.EventName == constants.UserLoginWebhookEvent {
		reqBody["auth_recipe"] = constants.AuthRecipeMethodMagicLinkLogin
	}

	requestBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Debug().Err(err).Msg("error marshalling requestBody obj")
		return nil, nil, err
	}

	// SSRF protection: resolve the host once and pin the dialer to the validated
	// IP so http.Client cannot be tricked into re-resolving (DNS rebinding
	// TOCTOU). Skipped only when tests explicitly set
	// SkipTestEndpointSSRFValidation.
	skipSSRF := p.Env == constants.TestEnv && p.Config.SkipTestEndpointSSRFValidation
	var client *http.Client
	if skipSSRF {
		client = &http.Client{Timeout: testEndpointHTTPTimeout}
	} else {
		client, err = validators.SafeHTTPClient(ctx, params.Endpoint, testEndpointHTTPTimeout)
		if err != nil {
			log.Debug().Err(err).Str("endpoint", params.Endpoint).Msg("endpoint URL rejected by SSRF filter")
			return nil, nil, fmt.Errorf("invalid endpoint: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, params.Endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Debug().Err(err).Msg("error creating post request")
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, val := range params.Headers {
		req.Header.Set(key, headerValueString(val))
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Msg("error making request")
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Debug().Err(err).Msg("error reading response")
		return nil, nil, err
	}

	statusCode := int64(resp.StatusCode)
	return &model.TestEndpointResponse{
		HTTPStatus: &statusCode,
		Response:   refs.NewStringRef(string(body)),
	}, nil, nil
}

// headerValueString coerces free-form JSON header values to strings without
// panicking. Mirrors the helper that previously lived in internal/graphql.
func headerValueString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

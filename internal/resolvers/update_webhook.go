package resolvers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/data_store/db"
	"github.com/authorizerdev/authorizer/internal/db"
	"github.com/authorizerdev/authorizer/internal/db/models"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
	log "github.com/sirupsen/logrus"
)

// UpdateWebhookResolver resolver for update webhook mutation
func UpdateWebhookResolver(ctx context.Context, params model.UpdateWebhookRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}
	webhook, err := db.Provider.GetWebhookByID(ctx, params.ID)
	if err != nil {
		log.Debug("failed to get webhook: ", err)
		return nil, err
	}
	headersString := ""
	if webhook.Headers != nil {
		headerBytes, err := json.Marshal(webhook.Headers)
		if err != nil {
			log.Debug("failed to marshall source headers: ", err)
		}
		headersString = string(headerBytes)
	}
	webhookDetails := &models.Webhook{
		ID:               webhook.ID,
		Key:              webhook.ID,
		EventName:        refs.StringValue(webhook.EventName),
		EventDescription: refs.StringValue(webhook.EventDescription),
		EndPoint:         refs.StringValue(webhook.Endpoint),
		Enabled:          refs.BoolValue(webhook.Enabled),
		Headers:          headersString,
		CreatedAt:        refs.Int64Value(webhook.CreatedAt),
	}
	if params.EventName != nil && webhookDetails.EventName != refs.StringValue(params.EventName) {
		if isValid := validators.IsValidWebhookEventName(refs.StringValue(params.EventName)); !isValid {
			log.Debug("invalid event name: ", refs.StringValue(params.EventName))
			return nil, fmt.Errorf("invalid event name %s", refs.StringValue(params.EventName))
		}
		webhookDetails.EventName = refs.StringValue(params.EventName)
	}
	if params.Endpoint != nil && webhookDetails.EndPoint != refs.StringValue(params.Endpoint) {
		if strings.TrimSpace(refs.StringValue(params.Endpoint)) == "" {
			log.Debug("empty endpoint not allowed")
			return nil, fmt.Errorf("empty endpoint not allowed")
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
			log.Debug("failed to marshall headers: ", err)
			return nil, err
		}

		webhookDetails.Headers = string(headerBytes)
	}
	_, err = db.Provider.UpdateWebhook(ctx, webhookDetails)
	if err != nil {
		return nil, err
	}
	return &model.Response{
		Message: `Webhook updated successfully.`,
	}, nil
}

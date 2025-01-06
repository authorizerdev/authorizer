package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// UpdateWebhook is the method to update webhook details
// Permission: authorizer:admin
func (s *service) UpdateWebhook(ctx context.Context, params *model.UpdateWebhookRequest) (*model.Response, error) {
	log := s.Log.With().Str("func", "UpdateWebhook").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !s.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}
	webhook, err := s.StorageProvider.GetWebhookByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("failed GetWebhookByID")
		return nil, err
	}
	headersString := ""
	if webhook.Headers != nil {
		headerBytes, err := json.Marshal(webhook.Headers)
		if err != nil {
			log.Debug().Err(err).Msg("failed to marshall headers")
			return nil, err
		}
		headersString = string(headerBytes)
	}
	webhookDetails := &schemas.Webhook{
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
			log.Debug().Str("event_name", refs.StringValue(params.EventName)).Msg("invalid event name")
			return nil, fmt.Errorf("invalid event name %s", refs.StringValue(params.EventName))
		}
		webhookDetails.EventName = refs.StringValue(params.EventName)
	}
	if params.Endpoint != nil && webhookDetails.EndPoint != refs.StringValue(params.Endpoint) {
		if strings.TrimSpace(refs.StringValue(params.Endpoint)) == "" {
			log.Debug().Msg("empty endpoint not allowed")
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
			log.Debug().Err(err).Msg("failed to marshall headers")
			return nil, err
		}

		webhookDetails.Headers = string(headerBytes)
	}
	_, err = s.StorageProvider.UpdateWebhook(ctx, webhookDetails)
	if err != nil {
		log.Debug().Err(err).Msg("failed UpdateWebhook")
		return nil, err
	}
	return &model.Response{
		Message: `Webhook updated successfully.`,
	}, nil
}

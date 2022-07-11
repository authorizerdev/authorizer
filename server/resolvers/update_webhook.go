package resolvers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
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

	webhookDetails := models.Webhook{
		ID:        webhook.ID,
		Key:       webhook.ID,
		EventName: utils.StringValue(webhook.EventName),
		EndPoint:  utils.StringValue(webhook.Endpoint),
		Enabled:   utils.BoolValue(webhook.Enabled),
		Headers:   headersString,
		CreatedAt: *webhook.CreatedAt,
	}

	if webhookDetails.EventName != utils.StringValue(params.EventName) {
		if isValid := validators.IsValidWebhookEventName(utils.StringValue(params.EventName)); !isValid {
			log.Debug("invalid event name: ", utils.StringValue(params.EventName))
			return nil, fmt.Errorf("invalid event name %s", utils.StringValue(params.EventName))
		}
		webhookDetails.EventName = utils.StringValue(params.EventName)
	}

	if webhookDetails.EndPoint != utils.StringValue(params.Endpoint) {
		webhookDetails.EventName = utils.StringValue(params.EventName)
	}

	if webhookDetails.Enabled != utils.BoolValue(params.Enabled) {
		webhookDetails.Enabled = utils.BoolValue(params.Enabled)
	}

	if params.Headers != nil {
		for key, val := range params.Headers {
			webhook.Headers[key] = val
		}

		headerBytes, err := json.Marshal(webhook.Headers)
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

package resolvers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/refs"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/authorizerdev/authorizer/server/validators"
	log "github.com/sirupsen/logrus"
)

// AddWebhookResolver resolver for add webhook mutation
func AddWebhookResolver(ctx context.Context, params model.AddWebhookRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}
	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}
	if !validators.IsValidWebhookEventName(params.EventName) {
		log.Debug("Invalid Event Name: ", params.EventName)
		return nil, fmt.Errorf("invalid event name %s", params.EventName)
	}
	if strings.TrimSpace(params.Endpoint) == "" {
		log.Debug("empty endpoint not allowed")
		return nil, fmt.Errorf("empty endpoint not allowed")
	}
	headerBytes, err := json.Marshal(params.Headers)
	if err != nil {
		return nil, err
	}

	if params.EventDescription == nil {
		params.EventDescription = refs.NewStringRef(strings.Join(strings.Split(params.EventName, "."), " "))
	}
	_, err = db.Provider.AddWebhook(ctx, models.Webhook{
		EventDescription: refs.StringValue(params.EventDescription),
		EventName:        params.EventName,
		EndPoint:         params.Endpoint,
		Enabled:          params.Enabled,
		Headers:          string(headerBytes),
	})
	if err != nil {
		log.Debug("Failed to add webhook: ", err)
		return nil, err
	}

	return &model.Response{
		Message: `Webhook added successfully`,
	}, nil
}

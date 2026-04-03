package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
	"github.com/authorizerdev/authorizer/internal/validators"
)

// AddWebhook is the method to add webhook.
// Permissions: authorizer:admin
func (g *graphqlProvider) AddWebhook(ctx context.Context, params *model.AddWebhookRequest) (*model.Response, error) {
	log := g.Log.With().Str("func", "AddWebhook").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}
	if !validators.IsValidWebhookEventName(params.EventName) {
		log.Debug().Str("EventName", params.EventName).Msg("Invalid Event Name")
		return nil, fmt.Errorf("invalid event name %s", params.EventName)
	}
	if strings.TrimSpace(params.Endpoint) == "" {
		log.Debug().Msg("endpoint is missing")
		return nil, fmt.Errorf("empty endpoint not allowed")
	}
	headerBytes, err := json.Marshal(params.Headers)
	if err != nil {
		return nil, err
	}

	if params.EventDescription == nil {
		params.EventDescription = refs.NewStringRef(strings.Join(strings.Split(params.EventName, "."), " "))
	}
	webhook, err := g.StorageProvider.AddWebhook(ctx, &schemas.Webhook{
		EventDescription: refs.StringValue(params.EventDescription),
		EventName:        params.EventName,
		EndPoint:         params.Endpoint,
		Enabled:          params.Enabled,
		Headers:          string(headerBytes),
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to add webhook in db")
		return nil, err
	}

	g.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminWebhookCreatedEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeWebhook,
		ResourceID:   webhook.ID,
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
	})
	return &model.Response{
		Message: `Webhook added successfully`,
	}, nil
}

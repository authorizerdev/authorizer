package service

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// DeleteWebhook is the method to delete a webhook.
// Permissions: authorizer:admin
func (s *service) DeleteWebhook(ctx context.Context, params *model.WebhookRequest) (*model.Response, error) {
	log := s.Log.With().Str("func", "DeleteWebhook").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	if !s.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	if params.ID == "" {
		log.Debug().Msg("Webhook ID required")
		return nil, fmt.Errorf("webhook ID required")
	}

	log = log.With().Str("webhookID", params.ID).Logger()

	webhook, err := s.StorageProvider.GetWebhookByID(ctx, params.ID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get webhook by ID")
		return nil, err
	}

	err = s.StorageProvider.DeleteWebhook(ctx, webhook)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to delete webhook")
		return nil, err
	}

	return &model.Response{
		Message: "Webhook deleted successfully",
	}, nil
}

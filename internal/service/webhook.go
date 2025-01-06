package service

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Webhook is the method to get webhook details
// Permission: authorizer:admin
func (s *service) Webhook(ctx context.Context, params *model.WebhookRequest) (*model.Webhook, error) {
	log := s.Log.With().Str("func", "Webhook").Logger()
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
	return webhook, nil
}

package service

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Webhooks is the method to list webhooks
// Permission: authorizer:admin
func (s *service) Webhooks(ctx context.Context, params *model.PaginatedInput) (*model.Webhooks, error) {
	log := s.Log.With().Str("func", "Webhooks").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !s.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(params)
	webhooks, err := s.StorageProvider.ListWebhook(ctx, pagination)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListWebhook")
		return nil, err
	}
	return webhooks, nil
}

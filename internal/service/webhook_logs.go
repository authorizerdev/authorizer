package service

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// WebhookLogs is the method to get webhook logs
// Permission: authorizer:admin
func (s *service) WebhookLogs(ctx context.Context, params *model.ListWebhookLogRequest) (*model.WebhookLogs, error) {
	log := s.Log.With().Str("func", "WebhookLogs").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !s.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	var pagination *model.Pagination
	var webhookID string

	if params != nil {
		pagination = utils.GetPagination(&model.PaginatedInput{
			Pagination: params.Pagination,
		})
		webhookID = refs.StringValue(params.WebhookID)
	} else {
		pagination = utils.GetPagination(nil)
		webhookID = ""
	}
	// TODO fix
	webhookLogs, err := s.StorageProvider.ListWebhookLogs(ctx, pagination, webhookID)
	if err != nil {
		log.Debug().Err(err).Msg("failed ListWebhookLogs")
		return nil, err
	}
	return webhookLogs, nil
}

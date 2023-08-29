package resolvers

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
	log "github.com/sirupsen/logrus"
)

// WebhooksResolver resolver for getting the list of webhooks based on pagination
func WebhooksResolver(ctx context.Context, params *model.PaginatedInput) (*model.Webhooks, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(params)
	webhooks, err := db.Provider.ListWebhook(ctx, pagination)
	if err != nil {
		log.Debug("failed to get webhooks: ", err)
		return nil, err
	}
	return webhooks, nil
}

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

// WebhookLogsResolver resolver for getting the list of webhook_logs based on pagination & webhook identifier
func WebhookLogsResolver(ctx context.Context, params model.ListWebhookLogRequest) (*model.WebhookLogs, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return nil, err
	}

	if !token.IsSuperAdmin(gc) {
		log.Debug("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}

	pagination := utils.GetPagination(&model.PaginatedInput{
		Pagination: params.Pagination,
	})

	webhookLogs, err := db.Provider.ListWebhookLogs(ctx, pagination, utils.StringValue(params.WebhookID))
	if err != nil {
		log.Debug("failed to get webhook logs: ", err)
		return nil, err
	}
	return webhookLogs, nil
}

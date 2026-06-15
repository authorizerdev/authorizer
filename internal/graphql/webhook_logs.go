package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// WebhookLogs delegates to the transport-agnostic service layer. Resolver is a
// thin transport adapter.
//
// Permission: authorizer:admin
func (g *graphqlProvider) WebhookLogs(ctx context.Context, params *model.ListWebhookLogRequest) (*model.WebhookLogs, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().WebhookLogs(ctx, service.MetaFromGin(gc), params)
	return res, err
}

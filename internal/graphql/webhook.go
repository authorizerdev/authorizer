package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Webhook delegates to the transport-agnostic service layer. Resolver is a thin
// transport adapter.
//
// Permission: authorizer:admin
func (g *graphqlProvider) Webhook(ctx context.Context, params *model.WebhookRequest) (*model.Webhook, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return nil, err
	}
	res, _, err := g.adminService().Webhook(ctx, service.MetaFromGin(gc), params)
	return res, err
}

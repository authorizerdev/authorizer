package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// UpdateWebhook delegates to the transport-agnostic service layer. Resolver is a
// thin transport adapter.
//
// Permission: authorizer:admin
func (g *graphqlProvider) UpdateWebhook(ctx context.Context, params *model.UpdateWebhookRequest) (*model.Response, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().UpdateWebhook(ctx, service.MetaFromGin(gc), params)
	return res, err
}

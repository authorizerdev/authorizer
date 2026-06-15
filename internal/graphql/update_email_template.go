package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// UpdateEmailTemplate delegates to the transport-agnostic service layer.
// Resolver is a thin transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) UpdateEmailTemplate(ctx context.Context, params *model.UpdateEmailTemplateRequest) (*model.Response, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().UpdateEmailTemplate(ctx, service.MetaFromGin(gc), params)
	return res, err
}

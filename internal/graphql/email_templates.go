package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// EmailTemplates delegates to the transport-agnostic service layer. Resolver is
// a thin transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) EmailTemplates(ctx context.Context, params *model.PaginatedRequest) (*model.EmailTemplates, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().EmailTemplates(ctx, service.MetaFromGin(gc), params)
	return res, err
}

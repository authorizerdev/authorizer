package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// FgaExpand delegates to the transport-agnostic service layer. Resolver is a
// thin transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) FgaExpand(ctx context.Context, params *model.FgaExpandInput) (*model.FgaExpandResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return nil, err
	}
	res, _, err := g.adminService().FgaExpand(ctx, service.MetaFromGin(gc), params)
	return res, err
}

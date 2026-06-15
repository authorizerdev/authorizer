package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// FgaWriteTuples delegates to the transport-agnostic service layer. Resolver is
// a thin transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) FgaWriteTuples(ctx context.Context, params *model.FgaWriteTuplesInput) (*model.Response, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().FgaWriteTuples(ctx, service.MetaFromGin(gc), params)
	return res, err
}

package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// FgaReset delegates to the transport-agnostic service layer. Resolver is a thin
// transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) FgaReset(ctx context.Context) (*model.Response, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().FgaReset(ctx, service.MetaFromGin(gc))
	return res, err
}

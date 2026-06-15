package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// FgaGetModel delegates to the transport-agnostic service layer. Resolver is a
// thin transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) FgaGetModel(ctx context.Context) (*model.FgaModel, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().FgaGetModel(ctx, service.MetaFromGin(gc))
	return res, err
}

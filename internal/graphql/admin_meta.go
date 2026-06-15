package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AdminMeta delegates to the transport-agnostic service layer. Resolver is a
// thin transport adapter — same pattern as Meta.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) AdminMeta(ctx context.Context) (*model.AdminMeta, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().AdminMeta(ctx, service.MetaFromGin(gc))
	return res, err
}

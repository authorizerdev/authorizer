package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Permissions delegates to the transport-agnostic service layer.
// Permissions: authenticated user.
func (g *graphqlProvider) Permissions(ctx context.Context) ([]*model.Permission, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.ServiceProvider.Permissions(ctx, service.MetaFromGin(gc))
	return res, err
}

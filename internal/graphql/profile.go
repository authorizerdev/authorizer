package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Profile delegates to the transport-agnostic service layer.
// Permissions: authenticated user.
func (g *graphqlProvider) Profile(ctx context.Context) (*model.User, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return nil, err
	}
	res, _, err := g.ServiceProvider.Profile(ctx, service.MetaFromGin(gc))
	return res, err
}

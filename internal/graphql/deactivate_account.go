package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// DeactivateAccount delegates to the transport-agnostic service layer.
// Permissions: authenticated user.
func (g *graphqlProvider) DeactivateAccount(ctx context.Context) (*model.Response, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, side, err := g.ServiceProvider.DeactivateAccount(ctx, service.MetaFromGin(gc))
	if err != nil {
		return nil, err
	}
	service.ApplyToGin(gc, side)
	return res, nil
}

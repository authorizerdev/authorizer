package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AdminSession delegates to the transport-agnostic service layer and applies
// the refreshed admin session cookie side-effect. Resolver is a thin transport
// adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) AdminSession(ctx context.Context) (*model.Response, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, side, err := g.adminService().AdminSession(ctx, service.MetaFromGin(gc))
	if err != nil {
		return nil, err
	}
	service.ApplyToGin(gc, side)
	return res, nil
}

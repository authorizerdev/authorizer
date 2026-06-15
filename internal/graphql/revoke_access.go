package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// RevokeAccess delegates to the transport-agnostic service layer. Resolver is a
// thin transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) RevokeAccess(ctx context.Context, params *model.UpdateAccessRequest) (*model.Response, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().RevokeAccess(ctx, service.MetaFromGin(gc), params)
	return res, err
}

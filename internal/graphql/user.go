package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// User delegates to the transport-agnostic service layer. Resolver is a thin
// transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) User(ctx context.Context, params *model.GetUserRequest) (*model.User, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().User(ctx, service.MetaFromGin(gc), params)
	return res, err
}

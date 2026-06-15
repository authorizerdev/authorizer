package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// TestEndpoint delegates to the transport-agnostic service layer. Resolver is a
// thin transport adapter.
//
// Permission: authorizer:admin
func (g *graphqlProvider) TestEndpoint(ctx context.Context, params *model.TestEndpointRequest) (*model.TestEndpointResponse, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().TestEndpoint(ctx, service.MetaFromGin(gc), params)
	return res, err
}

package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// VerificationRequests delegates to the transport-agnostic service layer.
// Resolver is a thin transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) VerificationRequests(ctx context.Context, params *model.PaginatedRequest) (*model.VerificationRequests, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.adminService().VerificationRequests(ctx, service.MetaFromGin(gc), params)
	return res, err
}

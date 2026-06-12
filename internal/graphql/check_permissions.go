package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// CheckPermissions delegates to the transport-agnostic service layer, which
// owns the subject trust gate and fail-closed semantics.
func (g *graphqlProvider) CheckPermissions(ctx context.Context, params *model.CheckPermissionsInput) (*model.CheckPermissionsResponse, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.ServiceProvider.CheckPermissions(ctx, service.MetaFromGin(gc), params)
	return res, err
}

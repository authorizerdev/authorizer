package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ListPermissions delegates to the transport-agnostic service layer, which
// owns the subject trust gate, result caps, and fail-closed semantics.
func (g *graphqlProvider) ListPermissions(ctx context.Context, params *model.ListPermissionsInput) (*model.ListPermissionsResponse, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.ServiceProvider.ListPermissions(ctx, service.MetaFromGin(gc), params)
	return res, err
}

package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Revoke delegates to the transport-agnostic service layer.
func (g *graphqlProvider) Revoke(ctx context.Context, params *model.OAuthRevokeRequest) (*model.Response, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.ServiceProvider.Revoke(ctx, service.MetaFromGin(gc), params)
	return res, err
}

package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ValidateJWTToken delegates to the transport-agnostic service layer.
func (g *graphqlProvider) ValidateJWTToken(ctx context.Context, params *model.ValidateJWTTokenRequest) (*model.ValidateJWTTokenResponse, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.ServiceProvider.ValidateJwtToken(ctx, service.MetaFromGin(gc), params)
	return res, err
}

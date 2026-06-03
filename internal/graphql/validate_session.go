package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ValidateSession delegates to the transport-agnostic service layer.
func (g *graphqlProvider) ValidateSession(ctx context.Context, params *model.ValidateSessionRequest) (*model.ValidateSessionResponse, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, _, err := g.ServiceProvider.ValidateSession(ctx, service.MetaFromGin(gc), params)
	return res, err
}

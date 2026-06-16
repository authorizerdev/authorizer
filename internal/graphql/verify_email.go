package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// VerifyEmail delegates to the transport-agnostic service layer.
// Permissions: none.
func (g *graphqlProvider) VerifyEmail(ctx context.Context, params *model.VerifyEmailRequest) (*model.AuthResponse, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, side, err := g.ServiceProvider.VerifyEmail(ctx, service.MetaFromGin(gc), params)
	if err != nil {
		return nil, err
	}
	service.ApplyToGin(gc, side)
	return res, nil
}

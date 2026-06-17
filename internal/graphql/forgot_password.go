package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ForgotPassword delegates to the transport-agnostic service layer.
// Permissions: none.
func (g *graphqlProvider) ForgotPassword(ctx context.Context, params *model.ForgotPasswordRequest) (*model.ForgotPasswordResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return nil, err
	}
	res, side, err := g.ServiceProvider.ForgotPassword(ctx, service.MetaFromGin(gc), params)
	if err != nil {
		return nil, err
	}
	service.ApplyToGin(gc, side)
	return res, nil
}

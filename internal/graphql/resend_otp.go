package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ResendOTP delegates to the transport-agnostic service layer.
// Permissions: none.
func (g *graphqlProvider) ResendOTP(ctx context.Context, params *model.ResendOTPRequest) (*model.Response, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, side, err := g.ServiceProvider.ResendOTP(ctx, service.MetaFromGin(gc), params)
	if err != nil {
		return nil, err
	}
	service.ApplyToGin(gc, side)
	return res, nil
}

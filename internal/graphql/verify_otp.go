package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// VerifyOTP delegates to the transport-agnostic service layer.
// Permissions: none.
func (g *graphqlProvider) VerifyOTP(ctx context.Context, params *model.VerifyOTPRequest) (*model.AuthResponse, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, side, err := g.ServiceProvider.VerifyOTP(ctx, service.MetaFromGin(gc), params)
	if err != nil {
		return nil, err
	}
	service.ApplyToGin(gc, side)
	return res, nil
}

package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// UpdateProfile delegates to the transport-agnostic service layer.
// Permissions: authenticated user.
func (g *graphqlProvider) UpdateProfile(ctx context.Context, params *model.UpdateProfileRequest) (*model.Response, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, side, err := g.ServiceProvider.UpdateProfile(ctx, service.MetaFromGin(gc), params)
	if err != nil {
		return nil, err
	}
	service.ApplyToGin(gc, side)
	return res, nil
}

package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Session delegates to the transport-agnostic service layer; the rotated
// session cookie is applied back to the gin response from side-effects.
func (g *graphqlProvider) Session(ctx context.Context, params *model.SessionQueryRequest) (*model.AuthResponse, error) {
	gc, _ := utils.GinContextFromContext(ctx)
	res, side, err := g.ServiceProvider.Session(ctx, service.MetaFromGin(gc), params)
	if err != nil {
		return nil, err
	}
	service.ApplyToGin(gc, side)
	return res, nil
}

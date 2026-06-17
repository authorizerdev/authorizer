package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// SignUp delegates to the transport-agnostic service layer. Resolvers in this
// package are thin transport adapters: pull gin.Context out of the GraphQL
// context, build RequestMetadata, call the service, apply any cookie
// side-effects back onto gin.
//
// Permission: none
func (g *graphqlProvider) SignUp(ctx context.Context, params *model.SignUpRequest) (*model.AuthResponse, error) {
	log := g.Log.With().Str("func", "SignUp").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, side, err := g.ServiceProvider.SignUp(ctx, service.MetaFromGin(gc), params)
	if err != nil {
		return nil, err
	}
	service.ApplyToGin(gc, side)
	return res, nil
}

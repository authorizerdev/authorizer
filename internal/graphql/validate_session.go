package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// ValidateSession delegates to the transport-agnostic service layer.
func (g *graphqlProvider) ValidateSession(ctx context.Context, params *model.ValidateSessionRequest) (*model.ValidateSessionResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.ServiceProvider.ValidateSession(ctx, service.MetaFromGin(gc), params)
	return res, err
}

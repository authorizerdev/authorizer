package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AddEmailTemplate delegates to the transport-agnostic service layer. Resolver
// is a thin transport adapter.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) AddEmailTemplate(ctx context.Context, params *model.AddEmailTemplateRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().AddEmailTemplate(ctx, service.MetaFromGin(gc), params)
	return res, err
}

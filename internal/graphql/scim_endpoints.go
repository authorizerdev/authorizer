package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// CreateScimEndpoint delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) CreateScimEndpoint(ctx context.Context, params *model.CreateScimEndpointRequest) (*model.CreateScimEndpointResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().CreateScimEndpoint(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// RotateScimToken delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) RotateScimToken(ctx context.Context, params *model.ScimEndpointRequest) (*model.CreateScimEndpointResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().RotateScimToken(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// DeleteScimEndpoint delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) DeleteScimEndpoint(ctx context.Context, params *model.ScimEndpointRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().DeleteScimEndpoint(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// ScimEndpoint delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) ScimEndpoint(ctx context.Context, params *model.ScimEndpointRequest) (*model.ScimEndpoint, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().ScimEndpoint(ctx, service.MetaFromGin(gc), params)
	return res, err
}

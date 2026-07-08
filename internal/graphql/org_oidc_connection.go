package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// CreateOrgOidcConnection delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) CreateOrgOidcConnection(ctx context.Context, params *model.CreateOrgOIDCConnectionRequest) (*model.OrgOIDCConnection, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().CreateOrgOIDCConnection(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// UpdateOrgOidcConnection delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) UpdateOrgOidcConnection(ctx context.Context, params *model.UpdateOrgOIDCConnectionRequest) (*model.OrgOIDCConnection, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().UpdateOrgOIDCConnection(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// DeleteOrgOidcConnection delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) DeleteOrgOidcConnection(ctx context.Context, params *model.OrgOIDCConnectionRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().DeleteOrgOIDCConnection(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// OrgOidcConnection delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) OrgOidcConnection(ctx context.Context, params *model.OrgOIDCConnectionRequest) (*model.OrgOIDCConnection, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().OrgOIDCConnection(ctx, service.MetaFromGin(gc), params)
	return res, err
}

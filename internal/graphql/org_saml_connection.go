package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// CreateOrgSamlConnection delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) CreateOrgSamlConnection(ctx context.Context, params *model.CreateOrgSAMLConnectionRequest) (*model.OrgSAMLConnection, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().CreateOrgSAMLConnection(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// UpdateOrgSamlConnection delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) UpdateOrgSamlConnection(ctx context.Context, params *model.UpdateOrgSAMLConnectionRequest) (*model.OrgSAMLConnection, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().UpdateOrgSAMLConnection(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// DeleteOrgSamlConnection delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) DeleteOrgSamlConnection(ctx context.Context, params *model.OrgSAMLConnectionRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().DeleteOrgSAMLConnection(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// OrgSamlConnection delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) OrgSamlConnection(ctx context.Context, params *model.OrgSAMLConnectionRequest) (*model.OrgSAMLConnection, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().OrgSAMLConnection(ctx, service.MetaFromGin(gc), params)
	return res, err
}

package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// CreateOrganization delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) CreateOrganization(ctx context.Context, params *model.CreateOrganizationRequest) (*model.Organization, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().CreateOrganization(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// UpdateOrganization delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) UpdateOrganization(ctx context.Context, params *model.UpdateOrganizationRequest) (*model.Organization, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().UpdateOrganization(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// DeleteOrganization delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) DeleteOrganization(ctx context.Context, params *model.OrganizationRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().DeleteOrganization(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// Organization delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) Organization(ctx context.Context, params *model.OrganizationRequest) (*model.Organization, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().Organization(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// Organizations delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) Organizations(ctx context.Context, params *model.ListOrganizationsRequest) (*model.Organizations, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().Organizations(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// AddOrgMember delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) AddOrgMember(ctx context.Context, params *model.AddOrgMemberRequest) (*model.OrgMember, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().AddOrgMember(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// RemoveOrgMember delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) RemoveOrgMember(ctx context.Context, params *model.RemoveOrgMemberRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().RemoveOrgMember(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// OrgMembers delegates to the transport-agnostic service layer.
//
// Permissions: authorizer:admin
func (g *graphqlProvider) OrgMembers(ctx context.Context, params *model.ListOrgMembersRequest) (*model.OrgMembers, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().OrgMembers(ctx, service.MetaFromGin(gc), params)
	return res, err
}

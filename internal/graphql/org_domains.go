package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// RequestOrgDomain delegates to the transport-agnostic service layer.
func (g *graphqlProvider) RequestOrgDomain(ctx context.Context, params *model.RequestOrgDomainRequest) (*model.OrgDomainChallenge, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().RequestOrgDomain(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// VerifyOrgDomain delegates to the transport-agnostic service layer.
func (g *graphqlProvider) VerifyOrgDomain(ctx context.Context, params *model.VerifyOrgDomainRequest) (*model.OrgDomain, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().VerifyOrgDomain(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// AddVerifiedOrgDomain delegates to the transport-agnostic service layer.
func (g *graphqlProvider) AddVerifiedOrgDomain(ctx context.Context, params *model.AddVerifiedOrgDomainRequest) (*model.OrgDomain, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().AddVerifiedOrgDomain(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// OrgDomains delegates to the transport-agnostic service layer.
func (g *graphqlProvider) OrgDomains(ctx context.Context, params *model.ListOrgDomainsRequest) (*model.OrgDomains, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().OrgDomains(ctx, service.MetaFromGin(gc), params)
	return res, err
}

// DeleteOrgDomain delegates to the transport-agnostic service layer.
func (g *graphqlProvider) DeleteOrgDomain(ctx context.Context, params *model.DeleteOrgDomainRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		g.Log.Debug().Err(err).Msg("failed to get gin context")
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().DeleteOrgDomain(ctx, service.MetaFromGin(gc), params)
	return res, err
}

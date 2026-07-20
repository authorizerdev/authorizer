package graphql

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// The SAML IdP admin resolvers delegate to the transport-agnostic admin service,
// which enforces super-admin / org-admin permissions per operation.

func (g *graphqlProvider) CreateSamlServiceProvider(ctx context.Context, params *model.CreateSAMLServiceProviderRequest) (*model.SAMLServiceProvider, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().CreateSAMLServiceProvider(ctx, service.MetaFromGin(gc), params)
	return res, err
}

func (g *graphqlProvider) UpdateSamlServiceProvider(ctx context.Context, params *model.UpdateSAMLServiceProviderRequest) (*model.SAMLServiceProvider, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().UpdateSAMLServiceProvider(ctx, service.MetaFromGin(gc), params)
	return res, err
}

func (g *graphqlProvider) DeleteSamlServiceProvider(ctx context.Context, params *model.SAMLServiceProviderRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().DeleteSAMLServiceProvider(ctx, service.MetaFromGin(gc), params)
	return res, err
}

func (g *graphqlProvider) SamlServiceProvider(ctx context.Context, params *model.SAMLServiceProviderRequest) (*model.SAMLServiceProvider, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().SAMLServiceProvider(ctx, service.MetaFromGin(gc), params)
	return res, err
}

func (g *graphqlProvider) ListSamlServiceProviders(ctx context.Context, params *model.ListSAMLServiceProvidersRequest) (*model.SAMLServiceProviders, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().ListSAMLServiceProviders(ctx, service.MetaFromGin(gc), params)
	return res, err
}

func (g *graphqlProvider) RotateSamlIdpCert(ctx context.Context, params *model.RotateSAMLIDPCertRequest) (*model.SAMLIDPKey, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().RotateSAMLIDPCert(ctx, service.MetaFromGin(gc), params)
	return res, err
}

func (g *graphqlProvider) RetireSamlIdpKey(ctx context.Context, params *model.RetireSAMLIDPKeyRequest) (*model.Response, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().RetireSAMLIDPKey(ctx, service.MetaFromGin(gc), params)
	return res, err
}

func (g *graphqlProvider) ListSamlIdpKeys(ctx context.Context, params *model.ListSAMLIDPKeysRequest) ([]*model.SAMLIDPKey, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().ListSAMLIDPKeys(ctx, service.MetaFromGin(gc), params)
	return res, err
}

func (g *graphqlProvider) ImportSamlSpMetadata(ctx context.Context, params *model.ImportSAMLSPMetadataRequest) (*model.SAMLSPMetadataParseResult, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		metrics.RecordSecurityEvent(metrics.SecurityEventGinContextMissing, "graphql")
		return nil, err
	}
	res, _, err := g.adminService().ImportSAMLSPMetadata(ctx, service.MetaFromGin(gc), params)
	return res, err
}

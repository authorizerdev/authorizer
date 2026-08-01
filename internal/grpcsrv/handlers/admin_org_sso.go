package handlers

import (
	"context"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/grpcsrv/transport"
)

// Verified-domain, per-org SSO connection, and inbound-SCIM admin RPCs. Thin
// delegations: the service layer owns authorization, including resolving an
// org id from the loaded resource rather than from caller input.

// --- projectors ---

func projectOrgDomain(d *model.OrgDomain) *authorizerv1.OrgDomain {
	if d == nil {
		return nil
	}
	return &authorizerv1.OrgDomain{
		Domain:     d.Domain,
		OrgId:      d.OrgID,
		VerifiedAt: d.VerifiedAt,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

func projectOrgDomainChallenge(c *model.OrgDomainChallenge) *authorizerv1.OrgDomainChallenge {
	if c == nil {
		return nil
	}
	return &authorizerv1.OrgDomainChallenge{
		Domain:      c.Domain,
		RecordType:  c.RecordType,
		RecordName:  c.RecordName,
		RecordValue: c.RecordValue,
	}
}

func projectOrgDomains(d *model.OrgDomains) *authorizerv1.OrgDomainsResponse {
	if d == nil {
		return &authorizerv1.OrgDomainsResponse{}
	}
	items := make([]*authorizerv1.OrgDomain, 0, len(d.OrgDomains))
	for _, item := range d.OrgDomains {
		items = append(items, projectOrgDomain(item))
	}
	return &authorizerv1.OrgDomainsResponse{
		OrgDomains: items,
		Pagination: projectPagination(d.Pagination),
	}
}

func projectOrgOidcConnection(c *model.OrgOIDCConnection) *authorizerv1.OrgOidcConnection {
	if c == nil {
		return nil
	}
	return &authorizerv1.OrgOidcConnection{
		Id:          c.ID,
		OrgId:       c.OrgID,
		Name:        c.Name,
		IssuerUrl:   c.IssuerURL,
		SsoClientId: c.SsoClientID,
		Scopes:      c.Scopes,
		RedirectUri: c.RedirectURI,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func projectOrgSamlConnection(c *model.OrgSAMLConnection) *authorizerv1.OrgSamlConnection {
	if c == nil {
		return nil
	}
	return &authorizerv1.OrgSamlConnection{
		Id:                c.ID,
		OrgId:             c.OrgID,
		Name:              c.Name,
		IdpEntityId:       c.IdpEntityID,
		IdpSsoUrl:         c.IdpSsoURL,
		SpEntityId:        c.SpEntityID,
		AcsUrl:            c.AcsURL,
		AttributeMapping:  c.AttributeMapping,
		AllowIdpInitiated: c.AllowIdpInitiated,
		IsActive:          c.IsActive,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

func projectScimEndpoint(e *model.ScimEndpoint) *authorizerv1.ScimEndpoint {
	if e == nil {
		return nil
	}
	return &authorizerv1.ScimEndpoint{
		Id:        e.ID,
		OrgId:     e.OrgID,
		Enabled:   e.Enabled,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

// --- verified domains ---

// RequestOrgDomain delegates to service.RequestOrgDomain.
func (h *AdminHandler) RequestOrgDomain(ctx context.Context, req *authorizerv1.RequestOrgDomainRequest) (*authorizerv1.RequestOrgDomainResponse, error) {
	res, _, err := h.Service.RequestOrgDomain(ctx, transport.MetaFromGRPC(ctx), &model.RequestOrgDomainRequest{
		OrgID:  req.GetOrgId(),
		Domain: req.GetDomain(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.RequestOrgDomainResponse{Challenge: projectOrgDomainChallenge(res)}, nil
}

// VerifyOrgDomain delegates to service.VerifyOrgDomain.
func (h *AdminHandler) VerifyOrgDomain(ctx context.Context, req *authorizerv1.VerifyOrgDomainRequest) (*authorizerv1.VerifyOrgDomainResponse, error) {
	res, _, err := h.Service.VerifyOrgDomain(ctx, transport.MetaFromGRPC(ctx), &model.VerifyOrgDomainRequest{
		OrgID:  req.GetOrgId(),
		Domain: req.GetDomain(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.VerifyOrgDomainResponse{OrgDomain: projectOrgDomain(res)}, nil
}

// AddVerifiedOrgDomain delegates to service.AddVerifiedOrgDomain.
func (h *AdminHandler) AddVerifiedOrgDomain(ctx context.Context, req *authorizerv1.AddVerifiedOrgDomainRequest) (*authorizerv1.AddVerifiedOrgDomainResponse, error) {
	res, _, err := h.Service.AddVerifiedOrgDomain(ctx, transport.MetaFromGRPC(ctx), &model.AddVerifiedOrgDomainRequest{
		OrgID:  req.GetOrgId(),
		Domain: req.GetDomain(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.AddVerifiedOrgDomainResponse{OrgDomain: projectOrgDomain(res)}, nil
}

// OrgDomains delegates to service.OrgDomains.
func (h *AdminHandler) OrgDomains(ctx context.Context, req *authorizerv1.OrgDomainsRequest) (*authorizerv1.OrgDomainsResponse, error) {
	res, _, err := h.Service.OrgDomains(ctx, transport.MetaFromGRPC(ctx), &model.ListOrgDomainsRequest{
		OrgID:      req.GetOrgId(),
		Pagination: modelPaginationRequest(req.GetPagination()),
	})
	if err != nil {
		return nil, err
	}
	return projectOrgDomains(res), nil
}

// DeleteOrgDomain delegates to service.DeleteOrgDomain.
func (h *AdminHandler) DeleteOrgDomain(ctx context.Context, req *authorizerv1.DeleteOrgDomainRequest) (*authorizerv1.DeleteOrgDomainResponse, error) {
	res, _, err := h.Service.DeleteOrgDomain(ctx, transport.MetaFromGRPC(ctx), &model.DeleteOrgDomainRequest{
		Domain: req.GetDomain(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeleteOrgDomainResponse{Message: res.Message}, nil
}

// --- OIDC connections ---

// CreateOrgOidcConnection delegates to service.CreateOrgOIDCConnection.
func (h *AdminHandler) CreateOrgOidcConnection(ctx context.Context, req *authorizerv1.CreateOrgOidcConnectionRequest) (*authorizerv1.CreateOrgOidcConnectionResponse, error) {
	res, _, err := h.Service.CreateOrgOIDCConnection(ctx, transport.MetaFromGRPC(ctx), &model.CreateOrgOIDCConnectionRequest{
		OrgID:        req.GetOrgId(),
		Name:         req.GetName(),
		IssuerURL:    req.GetIssuerUrl(),
		ClientID:     req.GetClientId(),
		ClientSecret: req.GetClientSecret(),
		Scopes:       req.Scopes,
		RedirectURI:  req.RedirectUri,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.CreateOrgOidcConnectionResponse{OrgOidcConnection: projectOrgOidcConnection(res)}, nil
}

// UpdateOrgOidcConnection delegates to service.UpdateOrgOIDCConnection.
func (h *AdminHandler) UpdateOrgOidcConnection(ctx context.Context, req *authorizerv1.UpdateOrgOidcConnectionRequest) (*authorizerv1.UpdateOrgOidcConnectionResponse, error) {
	res, _, err := h.Service.UpdateOrgOIDCConnection(ctx, transport.MetaFromGRPC(ctx), &model.UpdateOrgOIDCConnectionRequest{
		ID:           req.GetId(),
		Name:         req.Name,
		IssuerURL:    req.IssuerUrl,
		ClientID:     req.ClientId,
		ClientSecret: req.ClientSecret,
		Scopes:       req.Scopes,
		RedirectURI:  req.RedirectUri,
		IsActive:     req.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.UpdateOrgOidcConnectionResponse{OrgOidcConnection: projectOrgOidcConnection(res)}, nil
}

// DeleteOrgOidcConnection delegates to service.DeleteOrgOIDCConnection.
func (h *AdminHandler) DeleteOrgOidcConnection(ctx context.Context, req *authorizerv1.DeleteOrgOidcConnectionRequest) (*authorizerv1.DeleteOrgOidcConnectionResponse, error) {
	res, _, err := h.Service.DeleteOrgOIDCConnection(ctx, transport.MetaFromGRPC(ctx), &model.OrgOIDCConnectionRequest{
		ID:    req.Id,
		OrgID: req.OrgId,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeleteOrgOidcConnectionResponse{Message: res.Message}, nil
}

// GetOrgOidcConnection delegates to service.OrgOIDCConnection.
func (h *AdminHandler) GetOrgOidcConnection(ctx context.Context, req *authorizerv1.GetOrgOidcConnectionRequest) (*authorizerv1.GetOrgOidcConnectionResponse, error) {
	res, _, err := h.Service.OrgOIDCConnection(ctx, transport.MetaFromGRPC(ctx), &model.OrgOIDCConnectionRequest{
		ID:    req.Id,
		OrgID: req.OrgId,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.GetOrgOidcConnectionResponse{OrgOidcConnection: projectOrgOidcConnection(res)}, nil
}

// --- SAML connections ---

// CreateOrgSamlConnection delegates to service.CreateOrgSAMLConnection.
func (h *AdminHandler) CreateOrgSamlConnection(ctx context.Context, req *authorizerv1.CreateOrgSamlConnectionRequest) (*authorizerv1.CreateOrgSamlConnectionResponse, error) {
	res, _, err := h.Service.CreateOrgSAMLConnection(ctx, transport.MetaFromGRPC(ctx), &model.CreateOrgSAMLConnectionRequest{
		OrgID:             req.GetOrgId(),
		Name:              req.GetName(),
		IdpEntityID:       req.GetIdpEntityId(),
		IdpSsoURL:         req.GetIdpSsoUrl(),
		IdpCertificate:    req.GetIdpCertificate(),
		SpEntityID:        req.SpEntityId,
		AcsURL:            req.AcsUrl,
		AttributeMapping:  req.AttributeMapping,
		AllowIdpInitiated: req.AllowIdpInitiated,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.CreateOrgSamlConnectionResponse{OrgSamlConnection: projectOrgSamlConnection(res)}, nil
}

// UpdateOrgSamlConnection delegates to service.UpdateOrgSAMLConnection.
func (h *AdminHandler) UpdateOrgSamlConnection(ctx context.Context, req *authorizerv1.UpdateOrgSamlConnectionRequest) (*authorizerv1.UpdateOrgSamlConnectionResponse, error) {
	res, _, err := h.Service.UpdateOrgSAMLConnection(ctx, transport.MetaFromGRPC(ctx), &model.UpdateOrgSAMLConnectionRequest{
		ID:                req.GetId(),
		Name:              req.Name,
		IdpEntityID:       req.IdpEntityId,
		IdpSsoURL:         req.IdpSsoUrl,
		IdpCertificate:    req.IdpCertificate,
		SpEntityID:        req.SpEntityId,
		AcsURL:            req.AcsUrl,
		AttributeMapping:  req.AttributeMapping,
		AllowIdpInitiated: req.AllowIdpInitiated,
		IsActive:          req.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.UpdateOrgSamlConnectionResponse{OrgSamlConnection: projectOrgSamlConnection(res)}, nil
}

// DeleteOrgSamlConnection delegates to service.DeleteOrgSAMLConnection.
func (h *AdminHandler) DeleteOrgSamlConnection(ctx context.Context, req *authorizerv1.DeleteOrgSamlConnectionRequest) (*authorizerv1.DeleteOrgSamlConnectionResponse, error) {
	res, _, err := h.Service.DeleteOrgSAMLConnection(ctx, transport.MetaFromGRPC(ctx), &model.OrgSAMLConnectionRequest{
		ID:    req.Id,
		OrgID: req.OrgId,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeleteOrgSamlConnectionResponse{Message: res.Message}, nil
}

// GetOrgSamlConnection delegates to service.OrgSAMLConnection.
func (h *AdminHandler) GetOrgSamlConnection(ctx context.Context, req *authorizerv1.GetOrgSamlConnectionRequest) (*authorizerv1.GetOrgSamlConnectionResponse, error) {
	res, _, err := h.Service.OrgSAMLConnection(ctx, transport.MetaFromGRPC(ctx), &model.OrgSAMLConnectionRequest{
		ID:    req.Id,
		OrgID: req.OrgId,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.GetOrgSamlConnectionResponse{OrgSamlConnection: projectOrgSamlConnection(res)}, nil
}

// --- inbound SCIM ---

// CreateScimEndpoint delegates to service.CreateScimEndpoint. The returned
// token is the plaintext bearer credential, surfaced exactly once.
func (h *AdminHandler) CreateScimEndpoint(ctx context.Context, req *authorizerv1.CreateScimEndpointRequest) (*authorizerv1.CreateScimEndpointResponse, error) {
	res, _, err := h.Service.CreateScimEndpoint(ctx, transport.MetaFromGRPC(ctx), &model.CreateScimEndpointRequest{
		OrgID: req.GetOrgId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.CreateScimEndpointResponse{
		ScimEndpoint: projectScimEndpoint(res.ScimEndpoint),
		Token:        res.Token,
	}, nil
}

// RotateScimToken delegates to service.RotateScimToken. Reuses
// CreateScimEndpointResponse — the new token is surfaced exactly once.
func (h *AdminHandler) RotateScimToken(ctx context.Context, req *authorizerv1.RotateScimTokenRequest) (*authorizerv1.CreateScimEndpointResponse, error) {
	res, _, err := h.Service.RotateScimToken(ctx, transport.MetaFromGRPC(ctx), &model.ScimEndpointRequest{
		OrgID: req.GetOrgId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.CreateScimEndpointResponse{
		ScimEndpoint: projectScimEndpoint(res.ScimEndpoint),
		Token:        res.Token,
	}, nil
}

// DeleteScimEndpoint delegates to service.DeleteScimEndpoint.
func (h *AdminHandler) DeleteScimEndpoint(ctx context.Context, req *authorizerv1.DeleteScimEndpointRequest) (*authorizerv1.DeleteScimEndpointResponse, error) {
	res, _, err := h.Service.DeleteScimEndpoint(ctx, transport.MetaFromGRPC(ctx), &model.ScimEndpointRequest{
		OrgID: req.GetOrgId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeleteScimEndpointResponse{Message: res.Message}, nil
}

// GetScimEndpoint delegates to service.ScimEndpoint. The bearer token is never
// surfaced here.
func (h *AdminHandler) GetScimEndpoint(ctx context.Context, req *authorizerv1.GetScimEndpointRequest) (*authorizerv1.GetScimEndpointResponse, error) {
	res, _, err := h.Service.ScimEndpoint(ctx, transport.MetaFromGRPC(ctx), &model.ScimEndpointRequest{
		OrgID: req.GetOrgId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.GetScimEndpointResponse{ScimEndpoint: projectScimEndpoint(res)}, nil
}

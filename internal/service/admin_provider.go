package service

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/authctx"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// AdminProvider is the transport-agnostic API for Authorizer's super-admin
// operations (the `_`-prefixed GraphQL queries/mutations). The same concrete
// *provider that implements Provider also implements AdminProvider; the
// interface is split to keep the public Provider focused. Every method
// enforces super-admin auth via requireSuperAdmin except AdminLogin, which
// establishes it.
//
// During the staged migration this interface grows one domain group at a time
// (see specs/2026-06-15-authorizer-admin-service-plan.md). The compile-time
// assertion that *provider satisfies AdminProvider is added once every method
// has landed (final phase).
type AdminProvider interface {
	// Auth + meta.
	AdminLogin(ctx context.Context, meta RequestMetadata, params *model.AdminLoginRequest) (*model.Response, *ResponseSideEffects, error)
	AdminLogout(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error)
	AdminSession(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error)
	AdminMeta(ctx context.Context, meta RequestMetadata) (*model.AdminMeta, *ResponseSideEffects, error)

	// Users.
	Users(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.Users, *ResponseSideEffects, error)
	User(ctx context.Context, meta RequestMetadata, params *model.GetUserRequest) (*model.User, *ResponseSideEffects, error)
	UpdateUser(ctx context.Context, meta RequestMetadata, params *model.UpdateUserRequest) (*model.User, *ResponseSideEffects, error)
	DeleteUser(ctx context.Context, meta RequestMetadata, params *model.DeleteUserRequest) (*model.Response, *ResponseSideEffects, error)
	VerificationRequests(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.VerificationRequests, *ResponseSideEffects, error)

	// Access.
	RevokeAccess(ctx context.Context, meta RequestMetadata, params *model.UpdateAccessRequest) (*model.Response, *ResponseSideEffects, error)
	EnableAccess(ctx context.Context, meta RequestMetadata, params *model.UpdateAccessRequest) (*model.Response, *ResponseSideEffects, error)
	InviteMembers(ctx context.Context, meta RequestMetadata, params *model.InviteMemberRequest) (*model.InviteMembersResponse, *ResponseSideEffects, error)

	// Webhooks.
	AddWebhook(ctx context.Context, meta RequestMetadata, params *model.AddWebhookRequest) (*model.Response, *ResponseSideEffects, error)
	UpdateWebhook(ctx context.Context, meta RequestMetadata, params *model.UpdateWebhookRequest) (*model.Response, *ResponseSideEffects, error)
	DeleteWebhook(ctx context.Context, meta RequestMetadata, params *model.WebhookRequest) (*model.Response, *ResponseSideEffects, error)
	Webhook(ctx context.Context, meta RequestMetadata, params *model.WebhookRequest) (*model.Webhook, *ResponseSideEffects, error)
	Webhooks(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.Webhooks, *ResponseSideEffects, error)
	WebhookLogs(ctx context.Context, meta RequestMetadata, params *model.ListWebhookLogRequest) (*model.WebhookLogs, *ResponseSideEffects, error)
	TestEndpoint(ctx context.Context, meta RequestMetadata, params *model.TestEndpointRequest) (*model.TestEndpointResponse, *ResponseSideEffects, error)

	// Service accounts.
	CreateClient(ctx context.Context, meta RequestMetadata, params *model.CreateClientRequest) (*model.CreateClientResponse, *ResponseSideEffects, error)
	UpdateClient(ctx context.Context, meta RequestMetadata, params *model.UpdateClientRequest) (*model.Client, *ResponseSideEffects, error)
	DeleteClient(ctx context.Context, meta RequestMetadata, params *model.ClientRequest) (*model.Response, *ResponseSideEffects, error)
	RotateClientSecret(ctx context.Context, meta RequestMetadata, params *model.ClientRequest) (*model.CreateClientResponse, *ResponseSideEffects, error)
	Client(ctx context.Context, meta RequestMetadata, params *model.ClientRequest) (*model.Client, *ResponseSideEffects, error)
	Clients(ctx context.Context, meta RequestMetadata, params *model.ListClientsRequest) (*model.Clients, *ResponseSideEffects, error)

	// Trusted issuers.
	AddTrustedIssuer(ctx context.Context, meta RequestMetadata, params *model.AddTrustedIssuerRequest) (*model.TrustedIssuer, *ResponseSideEffects, error)
	UpdateTrustedIssuer(ctx context.Context, meta RequestMetadata, params *model.UpdateTrustedIssuerRequest) (*model.TrustedIssuer, *ResponseSideEffects, error)
	DeleteTrustedIssuer(ctx context.Context, meta RequestMetadata, params *model.TrustedIssuerRequest) (*model.Response, *ResponseSideEffects, error)
	TrustedIssuer(ctx context.Context, meta RequestMetadata, params *model.TrustedIssuerRequest) (*model.TrustedIssuer, *ResponseSideEffects, error)
	TrustedIssuers(ctx context.Context, meta RequestMetadata, params *model.ListTrustedIssuersRequest) (*model.TrustedIssuers, *ResponseSideEffects, error)

	// Per-org SSO OIDC connections (Authorizer as Relying Party).
	CreateOrgOIDCConnection(ctx context.Context, meta RequestMetadata, params *model.CreateOrgOIDCConnectionRequest) (*model.OrgOIDCConnection, *ResponseSideEffects, error)
	UpdateOrgOIDCConnection(ctx context.Context, meta RequestMetadata, params *model.UpdateOrgOIDCConnectionRequest) (*model.OrgOIDCConnection, *ResponseSideEffects, error)
	DeleteOrgOIDCConnection(ctx context.Context, meta RequestMetadata, params *model.OrgOIDCConnectionRequest) (*model.Response, *ResponseSideEffects, error)
	OrgOIDCConnection(ctx context.Context, meta RequestMetadata, params *model.OrgOIDCConnectionRequest) (*model.OrgOIDCConnection, *ResponseSideEffects, error)
	CreateOrgSAMLConnection(ctx context.Context, meta RequestMetadata, params *model.CreateOrgSAMLConnectionRequest) (*model.OrgSAMLConnection, *ResponseSideEffects, error)
	UpdateOrgSAMLConnection(ctx context.Context, meta RequestMetadata, params *model.UpdateOrgSAMLConnectionRequest) (*model.OrgSAMLConnection, *ResponseSideEffects, error)
	DeleteOrgSAMLConnection(ctx context.Context, meta RequestMetadata, params *model.OrgSAMLConnectionRequest) (*model.Response, *ResponseSideEffects, error)
	OrgSAMLConnection(ctx context.Context, meta RequestMetadata, params *model.OrgSAMLConnectionRequest) (*model.OrgSAMLConnection, *ResponseSideEffects, error)

	// Organizations and per-org membership.
	CreateOrganization(ctx context.Context, meta RequestMetadata, params *model.CreateOrganizationRequest) (*model.Organization, *ResponseSideEffects, error)
	UpdateOrganization(ctx context.Context, meta RequestMetadata, params *model.UpdateOrganizationRequest) (*model.Organization, *ResponseSideEffects, error)
	DeleteOrganization(ctx context.Context, meta RequestMetadata, params *model.OrganizationRequest) (*model.Response, *ResponseSideEffects, error)
	Organization(ctx context.Context, meta RequestMetadata, params *model.OrganizationRequest) (*model.Organization, *ResponseSideEffects, error)
	Organizations(ctx context.Context, meta RequestMetadata, params *model.ListOrganizationsRequest) (*model.Organizations, *ResponseSideEffects, error)
	AddOrgMember(ctx context.Context, meta RequestMetadata, params *model.AddOrgMemberRequest) (*model.OrgMember, *ResponseSideEffects, error)
	RemoveOrgMember(ctx context.Context, meta RequestMetadata, params *model.RemoveOrgMemberRequest) (*model.Response, *ResponseSideEffects, error)
	OrgMembers(ctx context.Context, meta RequestMetadata, params *model.ListOrgMembersRequest) (*model.OrgMembers, *ResponseSideEffects, error)

	// Per-org inbound SCIM 2.0 endpoints. The bearer token is revealed once.
	CreateScimEndpoint(ctx context.Context, meta RequestMetadata, params *model.CreateScimEndpointRequest) (*model.CreateScimEndpointResponse, *ResponseSideEffects, error)
	RotateScimToken(ctx context.Context, meta RequestMetadata, params *model.ScimEndpointRequest) (*model.CreateScimEndpointResponse, *ResponseSideEffects, error)
	DeleteScimEndpoint(ctx context.Context, meta RequestMetadata, params *model.ScimEndpointRequest) (*model.Response, *ResponseSideEffects, error)
	ScimEndpoint(ctx context.Context, meta RequestMetadata, params *model.ScimEndpointRequest) (*model.ScimEndpoint, *ResponseSideEffects, error)

	// Per-org verified domains for home-realm discovery. request/verify/list/
	// delete are org-admin gated; AddVerifiedOrgDomain is super-admin only.
	RequestOrgDomain(ctx context.Context, meta RequestMetadata, params *model.RequestOrgDomainRequest) (*model.OrgDomainChallenge, *ResponseSideEffects, error)
	VerifyOrgDomain(ctx context.Context, meta RequestMetadata, params *model.VerifyOrgDomainRequest) (*model.OrgDomain, *ResponseSideEffects, error)
	AddVerifiedOrgDomain(ctx context.Context, meta RequestMetadata, params *model.AddVerifiedOrgDomainRequest) (*model.OrgDomain, *ResponseSideEffects, error)
	OrgDomains(ctx context.Context, meta RequestMetadata, params *model.ListOrgDomainsRequest) (*model.OrgDomains, *ResponseSideEffects, error)
	DeleteOrgDomain(ctx context.Context, meta RequestMetadata, params *model.DeleteOrgDomainRequest) (*model.Response, *ResponseSideEffects, error)

	// Email templates.
	AddEmailTemplate(ctx context.Context, meta RequestMetadata, params *model.AddEmailTemplateRequest) (*model.Response, *ResponseSideEffects, error)
	UpdateEmailTemplate(ctx context.Context, meta RequestMetadata, params *model.UpdateEmailTemplateRequest) (*model.Response, *ResponseSideEffects, error)
	DeleteEmailTemplate(ctx context.Context, meta RequestMetadata, params *model.DeleteEmailTemplateRequest) (*model.Response, *ResponseSideEffects, error)
	EmailTemplates(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.EmailTemplates, *ResponseSideEffects, error)

	// Audit.
	AuditLogs(ctx context.Context, meta RequestMetadata, params *model.ListAuditLogRequest) (*model.AuditLogs, *ResponseSideEffects, error)

	// FGA admin.
	FgaWriteModel(ctx context.Context, meta RequestMetadata, params *model.FgaWriteModelInput) (*model.FgaModel, *ResponseSideEffects, error)
	FgaWriteTuples(ctx context.Context, meta RequestMetadata, params *model.FgaWriteTuplesInput) (*model.Response, *ResponseSideEffects, error)
	FgaDeleteTuples(ctx context.Context, meta RequestMetadata, params *model.FgaWriteTuplesInput) (*model.Response, *ResponseSideEffects, error)
	FgaReset(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error)
	FgaGetModel(ctx context.Context, meta RequestMetadata) (*model.FgaModel, *ResponseSideEffects, error)
	FgaReadTuples(ctx context.Context, meta RequestMetadata, params *model.FgaReadTuplesInput) (*model.FgaTuples, *ResponseSideEffects, error)
	FgaListUsers(ctx context.Context, meta RequestMetadata, params *model.FgaListUsersInput) (*model.FgaListUsersResponse, *ResponseSideEffects, error)
	FgaExpand(ctx context.Context, meta RequestMetadata, params *model.FgaExpandInput) (*model.FgaExpandResponse, *ResponseSideEffects, error)
}

// Compile-time guarantee that the concrete provider implements the full admin
// surface. Backed by real implementations plus transient stubs in
// admin_stubs.go during the staged migration.
var _ AdminProvider = (*provider)(nil)

// requireSuperAdmin enforces super-admin auth in the transport-agnostic layer.
// It reuses the gin-shim pattern (meta.Request carries the headers + cookies
// of the real request for the GraphQL/REST gin path, and a synthesized request
// for the pure-gRPC path — see transport.MetaFromGRPC) so the same
// TokenProvider.IsSuperAdmin check runs identically across every transport.
// Returns an Unauthenticated service error (mapped to gRPC Unauthenticated /
// HTTP 401) when the caller is not a super admin.
func (p *provider) requireSuperAdmin(ctx context.Context, meta RequestMetadata) error {
	if principal, ok := authctx.FromContext(ctx); ok && principal.IsSuperAdmin {
		return nil
	}
	gc := &gin.Context{Request: meta.Request}
	if !p.TokenProvider.IsSuperAdmin(gc) {
		return Unauthenticated("unauthorized")
	}
	return nil
}

// requireOrgAdmin enforces org-scoped admin auth for a single organization. It
// passes when EITHER the caller is a platform super-admin (the unchanged escape
// hatch, reusing requireSuperAdmin's exact positive path), OR the caller is an
// authenticated user who holds the reserved namespaced role
// constants.OrgRoleAdmin ("authorizer:org_admin") in an OrgMembership of orgID.
//
// It FAILS CLOSED: any error resolving the caller or their membership, a missing
// membership, or a membership lacking the reserved role all deny. The bare
// "admin" role is NOT accepted (see constants.OrgRoleAdmin) — only the
// namespaced role grants org-scoped admin rights.
//
// orgID is the tenant-isolation boundary and MUST be sourced correctly by the
// caller (design H2): for create/list ops it is the org being written
// (params.OrgID); for update/delete/get ops it is the OrgID of the target
// resource AFTER it has been loaded by id — never a caller-supplied org id
// checked before the load. Returns Unauthenticated to match requireSuperAdmin's
// error shape and to avoid leaking whether the target resource exists.
func (p *provider) requireOrgAdmin(ctx context.Context, meta RequestMetadata, orgID string) error {
	// Super-admin escape hatch — identical positive path to requireSuperAdmin.
	if p.requireSuperAdmin(ctx, meta) == nil {
		return nil
	}

	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return Unauthenticated("unauthorized")
	}

	tokenData, err := p.callerTokenData(ctx, meta)
	if err != nil || tokenData == nil || tokenData.UserID == "" {
		return Unauthenticated("unauthorized")
	}

	// Defense in depth: machine (client_credentials) tokens are never org
	// members. They are already rejected implicitly by the membership lookup
	// (their sub is a client id, not a user id), but reject them explicitly so
	// the intent survives any future change to how memberships are minted.
	if tokenData.LoginMethod == constants.AuthRecipeMethodServiceAccount {
		return Unauthenticated("unauthorized")
	}

	membership, err := p.StorageProvider.GetOrgMembership(ctx, orgID, tokenData.UserID)
	if err != nil || membership == nil {
		return Unauthenticated("unauthorized")
	}
	for _, role := range membership.ParsedRoles() {
		if role == constants.OrgRoleAdmin {
			return nil
		}
	}
	return Unauthenticated("unauthorized")
}

// rejectOrgIDMismatch is the confused-deputy guard for the id-or-org_id
// resolvers (design H2). Once a resource has been loaded by id, if the caller
// ALSO supplied an org_id that names a different org, deny: they are trying to
// act on another org's resource under their own org's authority. A nil/empty
// org_id (the common id-only call) is fine.
// maskNonSuperAdminError collapses a resource-resolution error (not-found,
// wrong-kind) into a uniform Unauthenticated for non-super-admin callers, so a
// tenant admin cannot use these ops as a cross-org existence/kind oracle
// (CWE-204): probing an arbitrary id must not reveal whether it exists or its
// kind in another org. Super-admins still get the precise error.
func (p *provider) maskNonSuperAdminError(ctx context.Context, meta RequestMetadata, err error) error {
	if p.requireSuperAdmin(ctx, meta) == nil {
		return err
	}
	return Unauthenticated("unauthorized")
}

func rejectOrgIDMismatch(paramOrgID *string, resourceOrgID string) error {
	if paramOrgID != nil {
		if v := strings.TrimSpace(*paramOrgID); v != "" && v != resourceOrgID {
			return Unauthenticated("unauthorized")
		}
	}
	return nil
}

package service

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/authctx"
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

package service

import (
	"context"

	"github.com/gin-gonic/gin"

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

// requireSuperAdmin enforces super-admin auth in the transport-agnostic layer.
// It reuses the gin-shim pattern (meta.Request carries the headers + cookies
// of the real request for the GraphQL/REST gin path, and a synthesized request
// for the pure-gRPC path — see transport.MetaFromGRPC) so the same
// TokenProvider.IsSuperAdmin check runs identically across every transport.
// Returns an Unauthenticated service error (mapped to gRPC Unauthenticated /
// HTTP 401) when the caller is not a super admin.
func (p *provider) requireSuperAdmin(meta RequestMetadata) error {
	gc := &gin.Context{Request: meta.Request}
	if !p.TokenProvider.IsSuperAdmin(gc) {
		return Unauthenticated("unauthorized")
	}
	return nil
}

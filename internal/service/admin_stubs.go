package service

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// This file holds transient stubs for admin operations whose service-layer
// migration has not yet landed in this PR. They exist so *provider satisfies
// the full AdminProvider interface from the first phase — which lets the gRPC
// server wire a non-nil admin service and keeps intermediate builds green. As
// each domain phase implements an op for real (in admin_users.go,
// admin_webhooks.go, etc.), its stub here is deleted. When this file is empty
// it is removed. See specs/2026-06-15-authorizer-admin-service-plan.md.

// adminNotImplemented is returned by not-yet-migrated admin stubs. Mapped to
// gRPC Internal / HTTP 500; never reached in a completed build.
func adminNotImplemented() error {
	return &Error{Kind: KindInternal, msg: "admin operation not yet implemented"}
}

// --- Users (Phase 2) ---

func (p *provider) Users(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.Users, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) User(ctx context.Context, meta RequestMetadata, params *model.GetUserRequest) (*model.User, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) UpdateUser(ctx context.Context, meta RequestMetadata, params *model.UpdateUserRequest) (*model.User, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) DeleteUser(ctx context.Context, meta RequestMetadata, params *model.DeleteUserRequest) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) VerificationRequests(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.VerificationRequests, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

// --- Access (Phase 3) ---

func (p *provider) RevokeAccess(ctx context.Context, meta RequestMetadata, params *model.UpdateAccessRequest) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) EnableAccess(ctx context.Context, meta RequestMetadata, params *model.UpdateAccessRequest) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) InviteMembers(ctx context.Context, meta RequestMetadata, params *model.InviteMemberRequest) (*model.InviteMembersResponse, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

// --- Webhooks (Phase 4) ---

func (p *provider) AddWebhook(ctx context.Context, meta RequestMetadata, params *model.AddWebhookRequest) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) UpdateWebhook(ctx context.Context, meta RequestMetadata, params *model.UpdateWebhookRequest) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) DeleteWebhook(ctx context.Context, meta RequestMetadata, params *model.WebhookRequest) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) Webhook(ctx context.Context, meta RequestMetadata, params *model.WebhookRequest) (*model.Webhook, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) Webhooks(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.Webhooks, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) WebhookLogs(ctx context.Context, meta RequestMetadata, params *model.ListWebhookLogRequest) (*model.WebhookLogs, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) TestEndpoint(ctx context.Context, meta RequestMetadata, params *model.TestEndpointRequest) (*model.TestEndpointResponse, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

// --- Email templates (Phase 5) ---

func (p *provider) AddEmailTemplate(ctx context.Context, meta RequestMetadata, params *model.AddEmailTemplateRequest) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) UpdateEmailTemplate(ctx context.Context, meta RequestMetadata, params *model.UpdateEmailTemplateRequest) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) DeleteEmailTemplate(ctx context.Context, meta RequestMetadata, params *model.DeleteEmailTemplateRequest) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) EmailTemplates(ctx context.Context, meta RequestMetadata, params *model.PaginatedRequest) (*model.EmailTemplates, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

// --- Audit (Phase 6) ---

func (p *provider) AuditLogs(ctx context.Context, meta RequestMetadata, params *model.ListAuditLogRequest) (*model.AuditLogs, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

// --- FGA admin (Phase 7) ---

func (p *provider) FgaWriteModel(ctx context.Context, meta RequestMetadata, params *model.FgaWriteModelInput) (*model.FgaModel, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) FgaWriteTuples(ctx context.Context, meta RequestMetadata, params *model.FgaWriteTuplesInput) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) FgaDeleteTuples(ctx context.Context, meta RequestMetadata, params *model.FgaWriteTuplesInput) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) FgaReset(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) FgaGetModel(ctx context.Context, meta RequestMetadata) (*model.FgaModel, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) FgaReadTuples(ctx context.Context, meta RequestMetadata, params *model.FgaReadTuplesInput) (*model.FgaTuples, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) FgaListUsers(ctx context.Context, meta RequestMetadata, params *model.FgaListUsersInput) (*model.FgaListUsersResponse, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

func (p *provider) FgaExpand(ctx context.Context, meta RequestMetadata, params *model.FgaExpandInput) (*model.FgaExpandResponse, *ResponseSideEffects, error) {
	return nil, nil, adminNotImplemented()
}

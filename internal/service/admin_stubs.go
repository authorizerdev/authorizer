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

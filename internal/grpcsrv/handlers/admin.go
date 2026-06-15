// Package handlers — admin.go hosts AdminHandler, the single gRPC service
// handler for Authorizer's admin (super-admin-only) API. It embeds the
// generated UnimplementedAuthorizerAdminServiceServer so any not-yet-migrated
// RPC returns codes.Unimplemented; methods are filled in one domain group at a
// time, each delegating to service.AdminProvider following the public
// AuthorizerHandler pattern.
package handlers

import (
	"context"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/grpcsrv/transport"
	"github.com/authorizerdev/authorizer/internal/service"
)

// AdminHandler implements authorizer.v1.AuthorizerAdminService. The single
// struct satisfies the entire admin service interface; methods become real
// one domain group at a time. Service is the transport-agnostic admin API.
type AdminHandler struct {
	authorizerv1.UnimplementedAuthorizerAdminServiceServer
	Service service.AdminProvider
}

// AdminLogin delegates to service.AdminLogin and lifts the admin session cookie
// side-effect onto the outgoing stream (grpc-gateway promotes it to Set-Cookie
// for REST callers). Public entry point — does not require an existing session.
func (h *AdminHandler) AdminLogin(ctx context.Context, req *authorizerv1.AdminLoginRequest) (*authorizerv1.AdminLoginResponse, error) {
	res, side, err := h.Service.AdminLogin(ctx, transport.MetaFromGRPC(ctx), &model.AdminLoginRequest{
		AdminSecret: req.AdminSecret,
	})
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return &authorizerv1.AdminLoginResponse{Message: res.Message}, nil
}

// AdminLogout delegates to service.AdminLogout and applies the cookie-clearing
// side-effect. Requires super-admin auth.
func (h *AdminHandler) AdminLogout(ctx context.Context, _ *authorizerv1.AdminLogoutRequest) (*authorizerv1.AdminLogoutResponse, error) {
	res, side, err := h.Service.AdminLogout(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return &authorizerv1.AdminLogoutResponse{Message: res.Message}, nil
}

// AdminSession delegates to service.AdminSession and applies the refreshed
// cookie side-effect. Requires super-admin auth.
func (h *AdminHandler) AdminSession(ctx context.Context, _ *authorizerv1.AdminSessionRequest) (*authorizerv1.AdminSessionResponse, error) {
	res, side, err := h.Service.AdminSession(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	_ = transport.ApplyToGRPC(ctx, side)
	return &authorizerv1.AdminSessionResponse{Message: res.Message}, nil
}

// AdminMeta delegates to service.AdminMeta and projects the result. Requires
// super-admin auth.
func (h *AdminHandler) AdminMeta(ctx context.Context, _ *authorizerv1.AdminMetaRequest) (*authorizerv1.AdminMetaResponse, error) {
	res, _, err := h.Service.AdminMeta(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	return &authorizerv1.AdminMetaResponse{AdminMeta: projectAdminMeta(res)}, nil
}

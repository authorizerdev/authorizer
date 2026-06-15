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

// Users delegates to service.Users and projects the paginated result. Requires
// super-admin auth.
func (h *AdminHandler) Users(ctx context.Context, req *authorizerv1.UsersRequest) (*authorizerv1.UsersResponse, error) {
	res, _, err := h.Service.Users(ctx, transport.MetaFromGRPC(ctx), modelPaginatedRequest(req.GetPagination()))
	if err != nil {
		return nil, err
	}
	return projectUsers(res), nil
}

// User delegates to service.User, resolving by id or email. Requires
// super-admin auth.
func (h *AdminHandler) User(ctx context.Context, req *authorizerv1.UserRequest) (*authorizerv1.UserResponse, error) {
	res, _, err := h.Service.User(ctx, transport.MetaFromGRPC(ctx), &model.GetUserRequest{
		ID:    optionalString(req.GetId()),
		Email: optionalString(req.GetEmail()),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.UserResponse{User: projectUser(res)}, nil
}

// UpdateUser delegates to service.UpdateUser and projects the updated user.
// Optional proto fields map 1:1 onto the model's nullable pointers. Requires
// super-admin auth.
func (h *AdminHandler) UpdateUser(ctx context.Context, req *authorizerv1.UpdateUserRequest) (*authorizerv1.UpdateUserResponse, error) {
	res, _, err := h.Service.UpdateUser(ctx, transport.MetaFromGRPC(ctx), &model.UpdateUserRequest{
		ID:                       req.GetId(),
		Email:                    req.Email,
		EmailVerified:            req.EmailVerified,
		GivenName:                req.GivenName,
		FamilyName:               req.FamilyName,
		MiddleName:               req.MiddleName,
		Nickname:                 req.Nickname,
		Gender:                   req.Gender,
		Birthdate:                req.Birthdate,
		PhoneNumber:              req.PhoneNumber,
		PhoneNumberVerified:      req.PhoneNumberVerified,
		Picture:                  req.Picture,
		Roles:                    protoToModelStringSlice(req.GetRoles()),
		IsMultiFactorAuthEnabled: req.IsMultiFactorAuthEnabled,
		AppData:                  appDataToMap(req.GetAppData()),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.UpdateUserResponse{User: projectUser(res)}, nil
}

// DeleteUser delegates to service.DeleteUser. Requires super-admin auth.
func (h *AdminHandler) DeleteUser(ctx context.Context, req *authorizerv1.DeleteUserRequest) (*authorizerv1.DeleteUserResponse, error) {
	res, _, err := h.Service.DeleteUser(ctx, transport.MetaFromGRPC(ctx), &model.DeleteUserRequest{
		Email: req.GetEmail(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeleteUserResponse{Message: res.Message}, nil
}

// VerificationRequests delegates to service.VerificationRequests and projects
// the paginated result. Requires super-admin auth.
func (h *AdminHandler) VerificationRequests(ctx context.Context, req *authorizerv1.VerificationRequestsRequest) (*authorizerv1.VerificationRequestsResponse, error) {
	res, _, err := h.Service.VerificationRequests(ctx, transport.MetaFromGRPC(ctx), modelPaginatedRequest(req.GetPagination()))
	if err != nil {
		return nil, err
	}
	return projectVerificationRequests(res), nil
}

// RevokeAccess delegates to service.RevokeAccess. Requires super-admin auth.
func (h *AdminHandler) RevokeAccess(ctx context.Context, req *authorizerv1.RevokeAccessRequest) (*authorizerv1.RevokeAccessResponse, error) {
	res, _, err := h.Service.RevokeAccess(ctx, transport.MetaFromGRPC(ctx), &model.UpdateAccessRequest{
		UserID: req.GetUserId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.RevokeAccessResponse{Message: res.Message}, nil
}

// EnableAccess delegates to service.EnableAccess. Requires super-admin auth.
func (h *AdminHandler) EnableAccess(ctx context.Context, req *authorizerv1.EnableAccessRequest) (*authorizerv1.EnableAccessResponse, error) {
	res, _, err := h.Service.EnableAccess(ctx, transport.MetaFromGRPC(ctx), &model.UpdateAccessRequest{
		UserID: req.GetUserId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.EnableAccessResponse{Message: res.Message}, nil
}

// InviteMembers delegates to service.InviteMembers and projects the invited
// users. Requires super-admin auth.
func (h *AdminHandler) InviteMembers(ctx context.Context, req *authorizerv1.InviteMembersRequest) (*authorizerv1.InviteMembersResponse, error) {
	res, _, err := h.Service.InviteMembers(ctx, transport.MetaFromGRPC(ctx), &model.InviteMemberRequest{
		Emails:      req.GetEmails(),
		RedirectURI: req.RedirectUri,
	})
	if err != nil {
		return nil, err
	}
	return projectInviteMembers(res), nil
}

// AddWebhook delegates to service.AddWebhook. Headers arrive as the shared
// AppData struct and unwrap to a free-form map. Requires super-admin auth.
func (h *AdminHandler) AddWebhook(ctx context.Context, req *authorizerv1.AddWebhookRequest) (*authorizerv1.AddWebhookResponse, error) {
	res, _, err := h.Service.AddWebhook(ctx, transport.MetaFromGRPC(ctx), &model.AddWebhookRequest{
		EventName:        req.GetEventName(),
		EventDescription: req.EventDescription,
		Endpoint:         req.GetEndpoint(),
		Enabled:          req.GetEnabled(),
		Headers:          appDataToMap(req.GetHeaders()),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.AddWebhookResponse{Message: res.Message}, nil
}

// UpdateWebhook delegates to service.UpdateWebhook. Optional proto fields map
// 1:1 onto the model's nullable pointers. Requires super-admin auth.
func (h *AdminHandler) UpdateWebhook(ctx context.Context, req *authorizerv1.UpdateWebhookRequest) (*authorizerv1.UpdateWebhookResponse, error) {
	res, _, err := h.Service.UpdateWebhook(ctx, transport.MetaFromGRPC(ctx), &model.UpdateWebhookRequest{
		ID:               req.GetId(),
		EventName:        req.EventName,
		EventDescription: req.EventDescription,
		Endpoint:         req.Endpoint,
		Enabled:          req.Enabled,
		Headers:          appDataToMap(req.GetHeaders()),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.UpdateWebhookResponse{Message: res.Message}, nil
}

// DeleteWebhook delegates to service.DeleteWebhook. Requires super-admin auth.
func (h *AdminHandler) DeleteWebhook(ctx context.Context, req *authorizerv1.DeleteWebhookRequest) (*authorizerv1.DeleteWebhookResponse, error) {
	res, _, err := h.Service.DeleteWebhook(ctx, transport.MetaFromGRPC(ctx), &model.WebhookRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeleteWebhookResponse{Message: res.Message}, nil
}

// GetWebhook delegates to service.Webhook and projects the result. Requires
// super-admin auth.
func (h *AdminHandler) GetWebhook(ctx context.Context, req *authorizerv1.GetWebhookRequest) (*authorizerv1.GetWebhookResponse, error) {
	res, _, err := h.Service.Webhook(ctx, transport.MetaFromGRPC(ctx), &model.WebhookRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.GetWebhookResponse{Webhook: projectWebhook(res)}, nil
}

// Webhooks delegates to service.Webhooks and projects the paginated result.
// Requires super-admin auth.
func (h *AdminHandler) Webhooks(ctx context.Context, req *authorizerv1.WebhooksRequest) (*authorizerv1.WebhooksResponse, error) {
	res, _, err := h.Service.Webhooks(ctx, transport.MetaFromGRPC(ctx), modelPaginatedRequest(req.GetPagination()))
	if err != nil {
		return nil, err
	}
	return projectWebhooks(res), nil
}

// WebhookLogs delegates to service.WebhookLogs and projects the paginated
// result. webhook_id is optional. Requires super-admin auth.
func (h *AdminHandler) WebhookLogs(ctx context.Context, req *authorizerv1.WebhookLogsRequest) (*authorizerv1.WebhookLogsResponse, error) {
	res, _, err := h.Service.WebhookLogs(ctx, transport.MetaFromGRPC(ctx), &model.ListWebhookLogRequest{
		Pagination: modelPaginationRequest(req.GetPagination()),
		WebhookID:  req.WebhookId,
	})
	if err != nil {
		return nil, err
	}
	return projectWebhookLogs(res), nil
}

// TestEndpoint delegates to service.TestEndpoint and projects the result. Makes
// an outbound HTTP call to the supplied endpoint. Requires super-admin auth.
func (h *AdminHandler) TestEndpoint(ctx context.Context, req *authorizerv1.TestEndpointRequest) (*authorizerv1.TestEndpointResponse, error) {
	res, _, err := h.Service.TestEndpoint(ctx, transport.MetaFromGRPC(ctx), &model.TestEndpointRequest{
		Endpoint:         req.GetEndpoint(),
		EventName:        req.GetEventName(),
		EventDescription: req.EventDescription,
		Headers:          appDataToMap(req.GetHeaders()),
	})
	if err != nil {
		return nil, err
	}
	return projectTestEndpointResponse(res), nil
}

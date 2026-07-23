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
	query := req.GetQuery()
	res, _, err := h.Service.Users(ctx, transport.MetaFromGRPC(ctx), &model.ListUsersRequest{
		Pagination: modelPaginationRequest(req.GetPagination()),
		Query:      &query,
	})
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
		ResetMfa:                 req.ResetMfa,
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
	res, _, err := h.Service.VerificationRequests(ctx, transport.MetaFromGRPC(ctx), modelPaginationRequest(req.GetPagination()))
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
	res, _, err := h.Service.Webhooks(ctx, transport.MetaFromGRPC(ctx), modelPaginationRequest(req.GetPagination()))
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

// AddEmailTemplate delegates to service.AddEmailTemplate. design is optional.
// Requires super-admin auth.
func (h *AdminHandler) AddEmailTemplate(ctx context.Context, req *authorizerv1.AddEmailTemplateRequest) (*authorizerv1.AddEmailTemplateResponse, error) {
	res, _, err := h.Service.AddEmailTemplate(ctx, transport.MetaFromGRPC(ctx), &model.AddEmailTemplateRequest{
		EventName: req.GetEventName(),
		Subject:   req.GetSubject(),
		Template:  req.GetTemplate(),
		Design:    req.Design,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.AddEmailTemplateResponse{Message: res.Message}, nil
}

// UpdateEmailTemplate delegates to service.UpdateEmailTemplate. Optional proto
// fields map 1:1 onto the model's nullable pointers. Requires super-admin auth.
func (h *AdminHandler) UpdateEmailTemplate(ctx context.Context, req *authorizerv1.UpdateEmailTemplateRequest) (*authorizerv1.UpdateEmailTemplateResponse, error) {
	res, _, err := h.Service.UpdateEmailTemplate(ctx, transport.MetaFromGRPC(ctx), &model.UpdateEmailTemplateRequest{
		ID:        req.GetId(),
		EventName: req.EventName,
		Template:  req.Template,
		Subject:   req.Subject,
		Design:    req.Design,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.UpdateEmailTemplateResponse{Message: res.Message}, nil
}

// DeleteEmailTemplate delegates to service.DeleteEmailTemplate. Requires
// super-admin auth.
func (h *AdminHandler) DeleteEmailTemplate(ctx context.Context, req *authorizerv1.DeleteEmailTemplateRequest) (*authorizerv1.DeleteEmailTemplateResponse, error) {
	res, _, err := h.Service.DeleteEmailTemplate(ctx, transport.MetaFromGRPC(ctx), &model.DeleteEmailTemplateRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeleteEmailTemplateResponse{Message: res.Message}, nil
}

// EmailTemplates delegates to service.EmailTemplates and projects the paginated
// result. Requires super-admin auth.
func (h *AdminHandler) EmailTemplates(ctx context.Context, req *authorizerv1.EmailTemplatesRequest) (*authorizerv1.EmailTemplatesResponse, error) {
	res, _, err := h.Service.EmailTemplates(ctx, transport.MetaFromGRPC(ctx), modelPaginationRequest(req.GetPagination()))
	if err != nil {
		return nil, err
	}
	return projectEmailTemplates(res), nil
}

// AuditLogs delegates to service.AuditLogs and projects the paginated result.
// All filter fields are optional. Requires super-admin auth.
func (h *AdminHandler) AuditLogs(ctx context.Context, req *authorizerv1.AuditLogsRequest) (*authorizerv1.AuditLogsResponse, error) {
	res, _, err := h.Service.AuditLogs(ctx, transport.MetaFromGRPC(ctx), &model.ListAuditLogRequest{
		Pagination:    modelPaginationRequest(req.GetPagination()),
		Action:        req.Action,
		ActorID:       req.ActorId,
		ResourceType:  req.ResourceType,
		ResourceID:    req.ResourceId,
		FromTimestamp: req.FromTimestamp,
		ToTimestamp:   req.ToTimestamp,
	})
	if err != nil {
		return nil, err
	}
	return projectAuditLogs(res), nil
}

// FgaGetModel delegates to service.FgaGetModel and projects the active model.
// Fails closed when no FGA engine is configured. Requires super-admin auth.
func (h *AdminHandler) FgaGetModel(ctx context.Context, _ *authorizerv1.FgaGetModelRequest) (*authorizerv1.FgaGetModelResponse, error) {
	res, _, err := h.Service.FgaGetModel(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	return &authorizerv1.FgaGetModelResponse{Model: projectFgaModel(res)}, nil
}

// FgaWriteModel delegates to service.FgaWriteModel and projects the new model.
// Fails closed when no FGA engine is configured. Requires super-admin auth.
func (h *AdminHandler) FgaWriteModel(ctx context.Context, req *authorizerv1.FgaWriteModelRequest) (*authorizerv1.FgaWriteModelResponse, error) {
	res, _, err := h.Service.FgaWriteModel(ctx, transport.MetaFromGRPC(ctx), &model.FgaWriteModelInput{
		Dsl: req.GetDsl(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.FgaWriteModelResponse{Model: projectFgaModel(res)}, nil
}

// FgaWriteTuples delegates to service.FgaWriteTuples. Fails closed when no FGA
// engine is configured. Requires super-admin auth.
func (h *AdminHandler) FgaWriteTuples(ctx context.Context, req *authorizerv1.FgaWriteTuplesRequest) (*authorizerv1.FgaWriteTuplesResponse, error) {
	res, _, err := h.Service.FgaWriteTuples(ctx, transport.MetaFromGRPC(ctx), &model.FgaWriteTuplesInput{
		Tuples: modelFgaTupleInputs(req.GetTuples()),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.FgaWriteTuplesResponse{Message: res.Message}, nil
}

// FgaDeleteTuples delegates to service.FgaDeleteTuples. Fails closed when no FGA
// engine is configured. Requires super-admin auth.
func (h *AdminHandler) FgaDeleteTuples(ctx context.Context, req *authorizerv1.FgaDeleteTuplesRequest) (*authorizerv1.FgaDeleteTuplesResponse, error) {
	res, _, err := h.Service.FgaDeleteTuples(ctx, transport.MetaFromGRPC(ctx), &model.FgaWriteTuplesInput{
		Tuples: modelFgaTupleInputs(req.GetTuples()),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.FgaDeleteTuplesResponse{Message: res.Message}, nil
}

// FgaReadTuples delegates to service.FgaReadTuples and projects the page. All
// filter fields are optional. Fails closed when no FGA engine is configured.
// Requires super-admin auth.
func (h *AdminHandler) FgaReadTuples(ctx context.Context, req *authorizerv1.FgaReadTuplesRequest) (*authorizerv1.FgaReadTuplesResponse, error) {
	res, _, err := h.Service.FgaReadTuples(ctx, transport.MetaFromGRPC(ctx), &model.FgaReadTuplesInput{
		User:              req.User,
		Relation:          req.Relation,
		Object:            req.Object,
		PageSize:          req.PageSize,
		ContinuationToken: req.ContinuationToken,
	})
	if err != nil {
		return nil, err
	}
	return projectFgaTuples(res), nil
}

// FgaListUsers delegates to service.FgaListUsers and projects the result. Fails
// closed when no FGA engine is configured. Requires super-admin auth.
func (h *AdminHandler) FgaListUsers(ctx context.Context, req *authorizerv1.FgaListUsersRequest) (*authorizerv1.FgaListUsersResponse, error) {
	res, _, err := h.Service.FgaListUsers(ctx, transport.MetaFromGRPC(ctx), &model.FgaListUsersInput{
		Object:   req.GetObject(),
		Relation: req.GetRelation(),
		UserType: req.GetUserType(),
	})
	if err != nil {
		return nil, err
	}
	return projectFgaListUsersResponse(res), nil
}

// FgaExpand delegates to service.FgaExpand and projects the result. Fails closed
// when no FGA engine is configured. Requires super-admin auth.
func (h *AdminHandler) FgaExpand(ctx context.Context, req *authorizerv1.FgaExpandRequest) (*authorizerv1.FgaExpandResponse, error) {
	res, _, err := h.Service.FgaExpand(ctx, transport.MetaFromGRPC(ctx), &model.FgaExpandInput{
		Relation: req.GetRelation(),
		Object:   req.GetObject(),
	})
	if err != nil {
		return nil, err
	}
	return projectFgaExpandResponse(res), nil
}

// FgaReset delegates to service.FgaReset. Refused while tuples still exist. Fails
// closed when no FGA engine is configured. Requires super-admin auth.
func (h *AdminHandler) FgaReset(ctx context.Context, _ *authorizerv1.FgaResetRequest) (*authorizerv1.FgaResetResponse, error) {
	res, _, err := h.Service.FgaReset(ctx, transport.MetaFromGRPC(ctx))
	if err != nil {
		return nil, err
	}
	return &authorizerv1.FgaResetResponse{Message: res.Message}, nil
}

// CreateClient delegates to service.CreateClient and returns the
// generated client secret exactly once (CreateClientResponse is the only
// admin message that carries a secret). Requires super-admin auth.
func (h *AdminHandler) CreateClient(ctx context.Context, req *authorizerv1.CreateClientRequest) (*authorizerv1.CreateClientResponse, error) {
	res, _, err := h.Service.CreateClient(ctx, transport.MetaFromGRPC(ctx), &model.CreateClientRequest{
		Name:          req.GetName(),
		Description:   req.Description,
		AllowedScopes: req.GetAllowedScopes(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.CreateClientResponse{
		Client:       projectClient(res.Client),
		ClientSecret: res.ClientSecret,
	}, nil
}

// UpdateClient delegates to service.UpdateClient. Optional proto
// fields map 1:1 onto the model's nullable pointers; the client secret is never
// touched. Requires super-admin auth.
func (h *AdminHandler) UpdateClient(ctx context.Context, req *authorizerv1.UpdateClientRequest) (*authorizerv1.UpdateClientResponse, error) {
	res, _, err := h.Service.UpdateClient(ctx, transport.MetaFromGRPC(ctx), &model.UpdateClientRequest{
		ID:            req.GetId(),
		Name:          req.Name,
		Description:   req.Description,
		AllowedScopes: req.GetAllowedScopes(),
		IsActive:      req.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.UpdateClientResponse{Client: projectClient(res)}, nil
}

// DeleteClient delegates to service.DeleteClient, which cascades
// to the account's trusted issuers. Requires super-admin auth.
func (h *AdminHandler) DeleteClient(ctx context.Context, req *authorizerv1.DeleteClientRequest) (*authorizerv1.DeleteClientResponse, error) {
	res, _, err := h.Service.DeleteClient(ctx, transport.MetaFromGRPC(ctx), &model.ClientRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeleteClientResponse{Message: res.Message}, nil
}

// RotateClientSecret delegates to service.RotateClientSecret and
// returns the new client secret exactly once (reusing CreateClientResponse).
// Requires super-admin auth.
func (h *AdminHandler) RotateClientSecret(ctx context.Context, req *authorizerv1.RotateClientSecretRequest) (*authorizerv1.CreateClientResponse, error) {
	res, _, err := h.Service.RotateClientSecret(ctx, transport.MetaFromGRPC(ctx), &model.ClientRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.CreateClientResponse{
		Client:       projectClient(res.Client),
		ClientSecret: res.ClientSecret,
	}, nil
}

// GetClient delegates to service.Client and projects the result.
// The client secret is never surfaced. Requires super-admin auth.
func (h *AdminHandler) GetClient(ctx context.Context, req *authorizerv1.GetClientRequest) (*authorizerv1.GetClientResponse, error) {
	res, _, err := h.Service.Client(ctx, transport.MetaFromGRPC(ctx), &model.ClientRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.GetClientResponse{Client: projectClient(res)}, nil
}

// Clients delegates to service.Clients and projects the
// paginated result. Client secrets are never surfaced. Requires super-admin auth.
func (h *AdminHandler) Clients(ctx context.Context, req *authorizerv1.ClientsRequest) (*authorizerv1.ClientsResponse, error) {
	res, _, err := h.Service.Clients(ctx, transport.MetaFromGRPC(ctx), &model.ListClientsRequest{
		Pagination: modelPaginationRequest(req.GetPagination()),
	})
	if err != nil {
		return nil, err
	}
	return projectClients(res), nil
}

// AddTrustedIssuer delegates to service.AddTrustedIssuer. subject_claim defaults
// to "sub" in the service layer when unset. Requires super-admin auth.
func (h *AdminHandler) AddTrustedIssuer(ctx context.Context, req *authorizerv1.AddTrustedIssuerRequest) (*authorizerv1.AddTrustedIssuerResponse, error) {
	res, _, err := h.Service.AddTrustedIssuer(ctx, transport.MetaFromGRPC(ctx), &model.AddTrustedIssuerRequest{
		ServiceAccountID:         req.GetServiceAccountId(),
		Name:                     req.GetName(),
		IssuerURL:                req.GetIssuerUrl(),
		KeySourceType:            req.GetKeySourceType(),
		JwksURL:                  req.JwksUrl,
		ExpectedAud:              req.GetExpectedAud(),
		SubjectClaim:             req.SubjectClaim,
		AllowedSubjects:          req.AllowedSubjects,
		IssuerType:               req.GetIssuerType(),
		SpiffeRefreshHintSeconds: req.SpiffeRefreshHintSeconds,
		EnableTokenReview:        req.EnableTokenReview,
		KubernetesAPIServerURL:   req.KubernetesApiServerUrl,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.AddTrustedIssuerResponse{TrustedIssuer: projectTrustedIssuer(res)}, nil
}

// UpdateTrustedIssuer delegates to service.UpdateTrustedIssuer. Optional proto
// fields map 1:1 onto the model's nullable pointers. Requires super-admin auth.
func (h *AdminHandler) UpdateTrustedIssuer(ctx context.Context, req *authorizerv1.UpdateTrustedIssuerRequest) (*authorizerv1.UpdateTrustedIssuerResponse, error) {
	res, _, err := h.Service.UpdateTrustedIssuer(ctx, transport.MetaFromGRPC(ctx), &model.UpdateTrustedIssuerRequest{
		ID:                       req.GetId(),
		Name:                     req.Name,
		JwksURL:                  req.JwksUrl,
		ExpectedAud:              req.ExpectedAud,
		AllowedSubjects:          req.AllowedSubjects,
		IsActive:                 req.IsActive,
		SpiffeRefreshHintSeconds: req.SpiffeRefreshHintSeconds,
		EnableTokenReview:        req.EnableTokenReview,
		KubernetesAPIServerURL:   req.KubernetesApiServerUrl,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.UpdateTrustedIssuerResponse{TrustedIssuer: projectTrustedIssuer(res)}, nil
}

// DeleteTrustedIssuer delegates to service.DeleteTrustedIssuer. Requires
// super-admin auth.
func (h *AdminHandler) DeleteTrustedIssuer(ctx context.Context, req *authorizerv1.DeleteTrustedIssuerRequest) (*authorizerv1.DeleteTrustedIssuerResponse, error) {
	res, _, err := h.Service.DeleteTrustedIssuer(ctx, transport.MetaFromGRPC(ctx), &model.TrustedIssuerRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeleteTrustedIssuerResponse{Message: res.Message}, nil
}

// GetTrustedIssuer delegates to service.TrustedIssuer and projects the result.
// Requires super-admin auth.
func (h *AdminHandler) GetTrustedIssuer(ctx context.Context, req *authorizerv1.GetTrustedIssuerRequest) (*authorizerv1.GetTrustedIssuerResponse, error) {
	res, _, err := h.Service.TrustedIssuer(ctx, transport.MetaFromGRPC(ctx), &model.TrustedIssuerRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.GetTrustedIssuerResponse{TrustedIssuer: projectTrustedIssuer(res)}, nil
}

// TrustedIssuers delegates to service.TrustedIssuers and projects the paginated
// result. service_account_id is optional. Requires super-admin auth.
func (h *AdminHandler) TrustedIssuers(ctx context.Context, req *authorizerv1.TrustedIssuersRequest) (*authorizerv1.TrustedIssuersResponse, error) {
	res, _, err := h.Service.TrustedIssuers(ctx, transport.MetaFromGRPC(ctx), &model.ListTrustedIssuersRequest{
		ServiceAccountID: req.ServiceAccountId,
		Pagination:       modelPaginationRequest(req.GetPagination()),
	})
	if err != nil {
		return nil, err
	}
	return projectTrustedIssuers(res), nil
}

// CreateSamlServiceProvider delegates to service.CreateSAMLServiceProvider. The
// optional proto fields map 1:1 onto the model's nullable pointers. Requires
// super-admin auth.
func (h *AdminHandler) CreateSamlServiceProvider(ctx context.Context, req *authorizerv1.CreateSamlServiceProviderRequest) (*authorizerv1.CreateSamlServiceProviderResponse, error) {
	res, _, err := h.Service.CreateSAMLServiceProvider(ctx, transport.MetaFromGRPC(ctx), &model.CreateSAMLServiceProviderRequest{
		OrgID:             req.GetOrgId(),
		Name:              req.GetName(),
		EntityID:          req.GetEntityId(),
		AcsURL:            req.GetAcsUrl(),
		SpCertPem:         req.SpCertPem,
		NameIDFormat:      req.NameIdFormat,
		MappedAttributes:  req.MappedAttributes,
		AllowIdpInitiated: req.AllowIdpInitiated,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.CreateSamlServiceProviderResponse{SamlServiceProvider: projectSamlServiceProvider(res)}, nil
}

// UpdateSamlServiceProvider delegates to service.UpdateSAMLServiceProvider.
// Optional proto fields map 1:1 onto the model's nullable pointers. Requires
// super-admin auth.
func (h *AdminHandler) UpdateSamlServiceProvider(ctx context.Context, req *authorizerv1.UpdateSamlServiceProviderRequest) (*authorizerv1.UpdateSamlServiceProviderResponse, error) {
	res, _, err := h.Service.UpdateSAMLServiceProvider(ctx, transport.MetaFromGRPC(ctx), &model.UpdateSAMLServiceProviderRequest{
		ID:                req.GetId(),
		Name:              req.Name,
		EntityID:          req.EntityId,
		AcsURL:            req.AcsUrl,
		SpCertPem:         req.SpCertPem,
		NameIDFormat:      req.NameIdFormat,
		MappedAttributes:  req.MappedAttributes,
		AllowIdpInitiated: req.AllowIdpInitiated,
		IsActive:          req.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.UpdateSamlServiceProviderResponse{SamlServiceProvider: projectSamlServiceProvider(res)}, nil
}

// DeleteSamlServiceProvider delegates to service.DeleteSAMLServiceProvider.
// Requires super-admin auth.
func (h *AdminHandler) DeleteSamlServiceProvider(ctx context.Context, req *authorizerv1.DeleteSamlServiceProviderRequest) (*authorizerv1.DeleteSamlServiceProviderResponse, error) {
	res, _, err := h.Service.DeleteSAMLServiceProvider(ctx, transport.MetaFromGRPC(ctx), &model.SAMLServiceProviderRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeleteSamlServiceProviderResponse{Message: res.Message}, nil
}

// GetSamlServiceProvider delegates to service.SAMLServiceProvider and projects
// the result. Requires super-admin auth.
func (h *AdminHandler) GetSamlServiceProvider(ctx context.Context, req *authorizerv1.GetSamlServiceProviderRequest) (*authorizerv1.GetSamlServiceProviderResponse, error) {
	res, _, err := h.Service.SAMLServiceProvider(ctx, transport.MetaFromGRPC(ctx), &model.SAMLServiceProviderRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.GetSamlServiceProviderResponse{SamlServiceProvider: projectSamlServiceProvider(res)}, nil
}

// ListSamlServiceProviders delegates to service.ListSAMLServiceProviders and
// projects the paginated result. Requires super-admin auth.
func (h *AdminHandler) ListSamlServiceProviders(ctx context.Context, req *authorizerv1.ListSamlServiceProvidersRequest) (*authorizerv1.ListSamlServiceProvidersResponse, error) {
	res, _, err := h.Service.ListSAMLServiceProviders(ctx, transport.MetaFromGRPC(ctx), &model.ListSAMLServiceProvidersRequest{
		OrgID:      req.GetOrgId(),
		Pagination: modelPaginationRequest(req.GetPagination()),
	})
	if err != nil {
		return nil, err
	}
	return projectSamlServiceProviders(res), nil
}

// RotateSamlIdpCert delegates to service.RotateSAMLIDPCert and projects the new
// current signing key. Requires super-admin auth.
func (h *AdminHandler) RotateSamlIdpCert(ctx context.Context, req *authorizerv1.RotateSamlIdpCertRequest) (*authorizerv1.RotateSamlIdpCertResponse, error) {
	res, _, err := h.Service.RotateSAMLIDPCert(ctx, transport.MetaFromGRPC(ctx), &model.RotateSAMLIDPCertRequest{
		OrgID: req.GetOrgId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.RotateSamlIdpCertResponse{SamlIdpKey: projectSamlIdpKey(res)}, nil
}

// RetireSamlIdpKey delegates to service.RetireSAMLIDPKey. Requires super-admin
// auth.
func (h *AdminHandler) RetireSamlIdpKey(ctx context.Context, req *authorizerv1.RetireSamlIdpKeyRequest) (*authorizerv1.RetireSamlIdpKeyResponse, error) {
	res, _, err := h.Service.RetireSAMLIDPKey(ctx, transport.MetaFromGRPC(ctx), &model.RetireSAMLIDPKeyRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.RetireSamlIdpKeyResponse{Message: res.Message}, nil
}

// ListSamlIdpKeys delegates to service.ListSAMLIDPKeys and projects the key set.
// Requires super-admin auth.
func (h *AdminHandler) ListSamlIdpKeys(ctx context.Context, req *authorizerv1.ListSamlIdpKeysRequest) (*authorizerv1.ListSamlIdpKeysResponse, error) {
	res, _, err := h.Service.ListSAMLIDPKeys(ctx, transport.MetaFromGRPC(ctx), &model.ListSAMLIDPKeysRequest{
		OrgID: req.GetOrgId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.ListSamlIdpKeysResponse{SamlIdpKeys: projectSamlIdpKeys(res)}, nil
}

// ImportSamlSpMetadata delegates to service.ImportSAMLSPMetadata. It parses
// pasted SP metadata XML and returns prefill fields; it creates no record.
// Requires super-admin auth.
func (h *AdminHandler) ImportSamlSpMetadata(ctx context.Context, req *authorizerv1.ImportSamlSpMetadataRequest) (*authorizerv1.ImportSamlSpMetadataResponse, error) {
	res, _, err := h.Service.ImportSAMLSPMetadata(ctx, transport.MetaFromGRPC(ctx), &model.ImportSAMLSPMetadataRequest{
		MetadataXML: req.GetMetadataXml(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.ImportSamlSpMetadataResponse{Result: projectSamlSpMetadataParseResult(res)}, nil
}

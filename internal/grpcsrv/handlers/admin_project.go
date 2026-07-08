// Package handlers — admin_project.go holds proto<->model projection helpers
// for the admin surface, mirroring project.go for the public surface. Each
// helper converts a GraphQL/storage model type returned by service.* into the
// proto wire type so the admin handler methods stay focused on delegation.
package handlers

import (
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// projectAdminMeta converts the GraphQL AdminMeta model into the proto message.
// The model already normalizes nil slices to empty, so the projection is a
// direct field copy.
func projectAdminMeta(m *model.AdminMeta) *authorizerv1.AdminMeta {
	if m == nil {
		return nil
	}
	return &authorizerv1.AdminMeta{
		Roles:          m.Roles,
		DefaultRoles:   m.DefaultRoles,
		ProtectedRoles: m.ProtectedRoles,
	}
}

// modelPaginatedRequest converts the proto PaginationRequest carried by admin
// list RPCs into the GraphQL model.PaginatedRequest consumed by the service
// layer. A nil proto pagination yields a nil request so service.GetPagination
// applies its defaults (page 1, default limit).
func modelPaginatedRequest(in *authorizerv1.PaginationRequest) *model.PaginatedRequest {
	if in == nil {
		return nil
	}
	out := &model.PaginatedRequest{Pagination: &model.PaginationRequest{}}
	if in.Page != 0 {
		page := in.Page
		out.Pagination.Page = &page
	}
	if in.Limit != 0 {
		limit := in.Limit
		out.Pagination.Limit = &limit
	}
	return out
}

// modelPaginationRequest converts the proto PaginationRequest into the GraphQL
// model.PaginationRequest (the inner pagination shape carried by
// ListWebhookLogRequest, as opposed to the PaginatedRequest wrapper). A nil
// proto pagination yields nil so service.GetPagination applies its defaults.
func modelPaginationRequest(in *authorizerv1.PaginationRequest) *model.PaginationRequest {
	if in == nil {
		return nil
	}
	out := &model.PaginationRequest{}
	if in.Page != 0 {
		page := in.Page
		out.Page = &page
	}
	if in.Limit != 0 {
		limit := in.Limit
		out.Limit = &limit
	}
	return out
}

// protoToModelStringSlice converts a proto repeated string field into the
// GraphQL model's []*string shape (used by UpdateUserRequest.Roles). A nil/empty
// input yields nil so "roles unset" is distinguishable from "roles cleared".
func protoToModelStringSlice(in []string) []*string {
	if len(in) == 0 {
		return nil
	}
	out := make([]*string, 0, len(in))
	for i := range in {
		v := in[i]
		out = append(out, &v)
	}
	return out
}

// projectPagination converts the GraphQL Pagination model into the shared proto
// Pagination message used by every admin list response.
func projectPagination(p *model.Pagination) *authorizerv1.Pagination {
	if p == nil {
		return nil
	}
	return &authorizerv1.Pagination{
		Limit:  p.Limit,
		Page:   p.Page,
		Offset: p.Offset,
		Total:  p.Total,
	}
}

// projectUsers converts the GraphQL Users model (a page of users plus its
// pagination cursor) into the proto UsersResponse.
func projectUsers(u *model.Users) *authorizerv1.UsersResponse {
	if u == nil {
		return &authorizerv1.UsersResponse{}
	}
	users := make([]*authorizerv1.User, 0, len(u.Users))
	for _, item := range u.Users {
		users = append(users, projectUser(item))
	}
	return &authorizerv1.UsersResponse{
		Users:      users,
		Pagination: projectPagination(u.Pagination),
	}
}

// projectVerificationRequest converts a single GraphQL VerificationRequest
// model into the proto message. Optional pointer fields collapse to zero
// values; EmitUnpopulated keeps them visible to REST clients.
func projectVerificationRequest(v *model.VerificationRequest) *authorizerv1.VerificationRequest {
	if v == nil {
		return nil
	}
	return &authorizerv1.VerificationRequest{
		Id:          v.ID,
		Identifier:  refs.StringValue(v.Identifier),
		Token:       refs.StringValue(v.Token),
		Email:       refs.StringValue(v.Email),
		Expires:     refs.Int64Value(v.Expires),
		CreatedAt:   refs.Int64Value(v.CreatedAt),
		UpdatedAt:   refs.Int64Value(v.UpdatedAt),
		Nonce:       refs.StringValue(v.Nonce),
		RedirectUri: refs.StringValue(v.RedirectURI),
	}
}

// projectVerificationRequests converts the GraphQL VerificationRequests model
// (a page plus its pagination cursor) into the proto response.
func projectVerificationRequests(v *model.VerificationRequests) *authorizerv1.VerificationRequestsResponse {
	if v == nil {
		return &authorizerv1.VerificationRequestsResponse{}
	}
	requests := make([]*authorizerv1.VerificationRequest, 0, len(v.VerificationRequests))
	for _, item := range v.VerificationRequests {
		requests = append(requests, projectVerificationRequest(item))
	}
	return &authorizerv1.VerificationRequestsResponse{
		VerificationRequests: requests,
		Pagination:           projectPagination(v.Pagination),
	}
}

// projectInviteMembers converts the GraphQL InviteMembersResponse model (a
// status message plus the list of newly invited users) into the proto response,
// reusing projectUser for each invited user.
func projectInviteMembers(r *model.InviteMembersResponse) *authorizerv1.InviteMembersResponse {
	if r == nil {
		return &authorizerv1.InviteMembersResponse{}
	}
	users := make([]*authorizerv1.User, 0, len(r.Users))
	for _, item := range r.Users {
		users = append(users, projectUser(item))
	}
	return &authorizerv1.InviteMembersResponse{
		Message: r.Message,
		Users:   users,
	}
}

// projectWebhook converts a single GraphQL Webhook model into the proto message.
// Optional pointer fields collapse to zero values; headers reuse the shared
// AppData wrapper. EmitUnpopulated keeps zero fields visible to REST clients.
func projectWebhook(w *model.Webhook) *authorizerv1.Webhook {
	if w == nil {
		return nil
	}
	return &authorizerv1.Webhook{
		Id:               w.ID,
		EventName:        refs.StringValue(w.EventName),
		EventDescription: refs.StringValue(w.EventDescription),
		Endpoint:         refs.StringValue(w.Endpoint),
		Enabled:          refs.BoolValue(w.Enabled),
		Headers:          mapToAppData(w.Headers),
		CreatedAt:        refs.Int64Value(w.CreatedAt),
		UpdatedAt:        refs.Int64Value(w.UpdatedAt),
	}
}

// projectWebhooks converts the GraphQL Webhooks model (a page plus its
// pagination cursor) into the proto response.
func projectWebhooks(w *model.Webhooks) *authorizerv1.WebhooksResponse {
	if w == nil {
		return &authorizerv1.WebhooksResponse{}
	}
	webhooks := make([]*authorizerv1.Webhook, 0, len(w.Webhooks))
	for _, item := range w.Webhooks {
		webhooks = append(webhooks, projectWebhook(item))
	}
	return &authorizerv1.WebhooksResponse{
		Webhooks:   webhooks,
		Pagination: projectPagination(w.Pagination),
	}
}

// projectWebhookLog converts a single GraphQL WebhookLog model into the proto
// message. Optional pointer fields collapse to zero values.
func projectWebhookLog(l *model.WebhookLog) *authorizerv1.WebhookLog {
	if l == nil {
		return nil
	}
	return &authorizerv1.WebhookLog{
		Id:         l.ID,
		HttpStatus: refs.Int64Value(l.HTTPStatus),
		Response:   refs.StringValue(l.Response),
		Request:    refs.StringValue(l.Request),
		WebhookId:  refs.StringValue(l.WebhookID),
		CreatedAt:  refs.Int64Value(l.CreatedAt),
		UpdatedAt:  refs.Int64Value(l.UpdatedAt),
	}
}

// projectWebhookLogs converts the GraphQL WebhookLogs model (a page plus its
// pagination cursor) into the proto response.
func projectWebhookLogs(l *model.WebhookLogs) *authorizerv1.WebhookLogsResponse {
	if l == nil {
		return &authorizerv1.WebhookLogsResponse{}
	}
	logs := make([]*authorizerv1.WebhookLog, 0, len(l.WebhookLogs))
	for _, item := range l.WebhookLogs {
		logs = append(logs, projectWebhookLog(item))
	}
	return &authorizerv1.WebhookLogsResponse{
		WebhookLogs: logs,
		Pagination:  projectPagination(l.Pagination),
	}
}

// projectTestEndpointResponse converts the GraphQL TestEndpointResponse model
// (the HTTP status and response body of a webhook test call) into the proto
// response. Optional pointer fields collapse to zero values.
func projectTestEndpointResponse(r *model.TestEndpointResponse) *authorizerv1.TestEndpointResponse {
	if r == nil {
		return &authorizerv1.TestEndpointResponse{}
	}
	return &authorizerv1.TestEndpointResponse{
		HttpStatus: refs.Int64Value(r.HTTPStatus),
		Response:   refs.StringValue(r.Response),
	}
}

// projectEmailTemplate converts a single GraphQL EmailTemplate model into the
// proto message. Optional pointer fields collapse to zero values;
// EmitUnpopulated keeps zero fields visible to REST clients.
func projectEmailTemplate(e *model.EmailTemplate) *authorizerv1.EmailTemplate {
	if e == nil {
		return nil
	}
	return &authorizerv1.EmailTemplate{
		Id:        e.ID,
		EventName: e.EventName,
		Template:  e.Template,
		Design:    e.Design,
		Subject:   e.Subject,
		CreatedAt: refs.Int64Value(e.CreatedAt),
		UpdatedAt: refs.Int64Value(e.UpdatedAt),
	}
}

// projectEmailTemplates converts the GraphQL EmailTemplates model (a page plus
// its pagination cursor) into the proto response.
func projectEmailTemplates(e *model.EmailTemplates) *authorizerv1.EmailTemplatesResponse {
	if e == nil {
		return &authorizerv1.EmailTemplatesResponse{}
	}
	templates := make([]*authorizerv1.EmailTemplate, 0, len(e.EmailTemplates))
	for _, item := range e.EmailTemplates {
		templates = append(templates, projectEmailTemplate(item))
	}
	return &authorizerv1.EmailTemplatesResponse{
		EmailTemplates: templates,
		Pagination:     projectPagination(e.Pagination),
	}
}

// projectAuditLog converts a single GraphQL AuditLog model into the proto
// message. Optional pointer fields collapse to zero values; EmitUnpopulated
// keeps zero fields visible to REST clients.
func projectAuditLog(a *model.AuditLog) *authorizerv1.AuditLog {
	if a == nil {
		return nil
	}
	return &authorizerv1.AuditLog{
		Id:           a.ID,
		ActorId:      refs.StringValue(a.ActorID),
		ActorType:    refs.StringValue(a.ActorType),
		ActorEmail:   refs.StringValue(a.ActorEmail),
		Action:       refs.StringValue(a.Action),
		ResourceType: refs.StringValue(a.ResourceType),
		ResourceId:   refs.StringValue(a.ResourceID),
		IpAddress:    refs.StringValue(a.IPAddress),
		UserAgent:    refs.StringValue(a.UserAgent),
		Metadata:     refs.StringValue(a.Metadata),
		CreatedAt:    refs.Int64Value(a.CreatedAt),
	}
}

// projectAuditLogs converts the GraphQL AuditLogs model (a page plus its
// pagination cursor) into the proto response.
func projectAuditLogs(a *model.AuditLogs) *authorizerv1.AuditLogsResponse {
	if a == nil {
		return &authorizerv1.AuditLogsResponse{}
	}
	logs := make([]*authorizerv1.AuditLog, 0, len(a.AuditLogs))
	for _, item := range a.AuditLogs {
		logs = append(logs, projectAuditLog(item))
	}
	return &authorizerv1.AuditLogsResponse{
		AuditLogs:  logs,
		Pagination: projectPagination(a.Pagination),
	}
}

// modelFgaTupleInputs converts proto FgaTupleInput messages (the shared request
// shape for tuple writes/deletes) into the GraphQL model's []*FgaTupleInput. A
// nil/empty input yields nil so the service layer's "at least one tuple" guard
// fires consistently across transports.
func modelFgaTupleInputs(in []*authorizerv1.FgaTupleInput) []*model.FgaTupleInput {
	if len(in) == 0 {
		return nil
	}
	out := make([]*model.FgaTupleInput, 0, len(in))
	for _, t := range in {
		if t == nil {
			continue
		}
		out = append(out, &model.FgaTupleInput{
			User:     t.GetUser(),
			Relation: t.GetRelation(),
			Object:   t.GetObject(),
		})
	}
	return out
}

// projectFgaModel converts the GraphQL FgaModel model (model id + DSL) into the
// proto message. An empty model is the valid "no model written yet" state.
func projectFgaModel(m *model.FgaModel) *authorizerv1.FgaModel {
	if m == nil {
		return nil
	}
	return &authorizerv1.FgaModel{
		Id:  m.ID,
		Dsl: m.Dsl,
	}
}

// projectFgaTuples converts the GraphQL FgaTuples model (a page of persisted
// tuples plus its continuation token) into the proto response.
func projectFgaTuples(t *model.FgaTuples) *authorizerv1.FgaReadTuplesResponse {
	if t == nil {
		return &authorizerv1.FgaReadTuplesResponse{}
	}
	tuples := make([]*authorizerv1.FgaTuple, 0, len(t.Tuples))
	for _, item := range t.Tuples {
		if item == nil {
			continue
		}
		tuples = append(tuples, &authorizerv1.FgaTuple{
			User:     item.User,
			Relation: item.Relation,
			Object:   item.Object,
		})
	}
	return &authorizerv1.FgaReadTuplesResponse{
		Tuples:            tuples,
		ContinuationToken: t.ContinuationToken,
	}
}

// projectFgaListUsersResponse converts the GraphQL FgaListUsersResponse model
// (the fully-qualified user ids that hold a relation on an object) into the proto
// response.
func projectFgaListUsersResponse(r *model.FgaListUsersResponse) *authorizerv1.FgaListUsersResponse {
	if r == nil {
		return &authorizerv1.FgaListUsersResponse{}
	}
	return &authorizerv1.FgaListUsersResponse{Users: r.Users}
}

// projectFgaExpandResponse converts the GraphQL FgaExpandResponse model (the
// relationship/userset tree as a JSON string) into the proto response.
func projectFgaExpandResponse(r *model.FgaExpandResponse) *authorizerv1.FgaExpandResponse {
	if r == nil {
		return &authorizerv1.FgaExpandResponse{}
	}
	return &authorizerv1.FgaExpandResponse{Tree: r.Tree}
}

// projectClient converts a single GraphQL Client model into the
// proto message. There is deliberately NO client-secret field on the proto
// Client: the plaintext secret is surfaced only by
// CreateClientResponse, so no get/list/update path can leak it.
func projectClient(s *model.Client) *authorizerv1.Client {
	if s == nil {
		return nil
	}
	return &authorizerv1.Client{
		Id:            s.ID,
		Name:          s.Name,
		Description:   refs.StringValue(s.Description),
		AllowedScopes: s.AllowedScopes,
		IsActive:      s.IsActive,
		CreatedAt:     refs.Int64Value(s.CreatedAt),
		UpdatedAt:     refs.Int64Value(s.UpdatedAt),
	}
}

// projectClients converts the GraphQL Clients model (a page plus
// its pagination cursor) into the proto response.
func projectClients(s *model.Clients) *authorizerv1.ClientsResponse {
	if s == nil {
		return &authorizerv1.ClientsResponse{}
	}
	accounts := make([]*authorizerv1.Client, 0, len(s.Clients))
	for _, item := range s.Clients {
		accounts = append(accounts, projectClient(item))
	}
	return &authorizerv1.ClientsResponse{
		Clients:    accounts,
		Pagination: projectPagination(s.Pagination),
	}
}

// projectTrustedIssuer converts a single GraphQL TrustedIssuer model into the
// proto message. Optional pointer fields collapse to zero values; the issuer
// references its parent via service_account_id.
func projectTrustedIssuer(t *model.TrustedIssuer) *authorizerv1.TrustedIssuer {
	if t == nil {
		return nil
	}
	return &authorizerv1.TrustedIssuer{
		Id:                       t.ID,
		ServiceAccountId:         t.ServiceAccountID,
		Name:                     t.Name,
		IssuerUrl:                t.IssuerURL,
		KeySourceType:            t.KeySourceType,
		JwksUrl:                  refs.StringValue(t.JwksURL),
		ExpectedAud:              t.ExpectedAud,
		SubjectClaim:             t.SubjectClaim,
		AllowedSubjects:          refs.StringValue(t.AllowedSubjects),
		IssuerType:               t.IssuerType,
		IsActive:                 t.IsActive,
		SpiffeRefreshHintSeconds: refs.Int64Value(t.SpiffeRefreshHintSeconds),
		CreatedAt:                refs.Int64Value(t.CreatedAt),
		UpdatedAt:                refs.Int64Value(t.UpdatedAt),
	}
}

// projectTrustedIssuers converts the GraphQL TrustedIssuers model (a page plus
// its pagination cursor) into the proto response.
func projectTrustedIssuers(t *model.TrustedIssuers) *authorizerv1.TrustedIssuersResponse {
	if t == nil {
		return &authorizerv1.TrustedIssuersResponse{}
	}
	issuers := make([]*authorizerv1.TrustedIssuer, 0, len(t.TrustedIssuers))
	for _, item := range t.TrustedIssuers {
		issuers = append(issuers, projectTrustedIssuer(item))
	}
	return &authorizerv1.TrustedIssuersResponse{
		TrustedIssuers: issuers,
		Pagination:     projectPagination(t.Pagination),
	}
}

// Package handlers — admin_project.go holds proto<->model projection helpers
// for the admin surface, mirroring project.go for the public surface. Each
// helper converts a GraphQL/storage model type returned by service.* into the
// proto wire type so the admin handler methods stay focused on delegation.
package handlers

import (
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"

	commonv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/common/v1"
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
func modelPaginatedRequest(in *commonv1.PaginationRequest) *model.PaginatedRequest {
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
func projectPagination(p *model.Pagination) *commonv1.Pagination {
	if p == nil {
		return nil
	}
	return &commonv1.Pagination{
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

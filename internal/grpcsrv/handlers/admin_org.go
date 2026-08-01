package handlers

import (
	"context"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/grpcsrv/transport"
)

// Organization / membership admin RPCs. Each is a thin delegation to the
// service layer, which owns the authorization decision — including the
// load-then-check discipline that sources an org id from the resource rather
// than from caller input. Nothing here re-implements or relaxes that.

// CreateOrganization delegates to service.CreateOrganization. Requires
// super-admin auth.
func (h *AdminHandler) CreateOrganization(ctx context.Context, req *authorizerv1.CreateOrganizationRequest) (*authorizerv1.CreateOrganizationResponse, error) {
	res, _, err := h.Service.CreateOrganization(ctx, transport.MetaFromGRPC(ctx), &model.CreateOrganizationRequest{
		Name:        req.GetName(),
		DisplayName: req.DisplayName,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.CreateOrganizationResponse{Organization: projectOrganization(res)}, nil
}

// UpdateOrganization delegates to service.UpdateOrganization. Optional proto
// fields map 1:1 onto the model's nullable pointers, so an omitted field leaves
// the stored value untouched.
func (h *AdminHandler) UpdateOrganization(ctx context.Context, req *authorizerv1.UpdateOrganizationRequest) (*authorizerv1.UpdateOrganizationResponse, error) {
	res, _, err := h.Service.UpdateOrganization(ctx, transport.MetaFromGRPC(ctx), &model.UpdateOrganizationRequest{
		ID:          req.GetId(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Enabled:     req.Enabled,
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.UpdateOrganizationResponse{Organization: projectOrganization(res)}, nil
}

// DeleteOrganization delegates to service.DeleteOrganization.
func (h *AdminHandler) DeleteOrganization(ctx context.Context, req *authorizerv1.DeleteOrganizationRequest) (*authorizerv1.DeleteOrganizationResponse, error) {
	res, _, err := h.Service.DeleteOrganization(ctx, transport.MetaFromGRPC(ctx), &model.OrganizationRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.DeleteOrganizationResponse{Message: res.Message}, nil
}

// GetOrganization delegates to service.Organization.
func (h *AdminHandler) GetOrganization(ctx context.Context, req *authorizerv1.GetOrganizationRequest) (*authorizerv1.GetOrganizationResponse, error) {
	res, _, err := h.Service.Organization(ctx, transport.MetaFromGRPC(ctx), &model.OrganizationRequest{
		ID: req.GetId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.GetOrganizationResponse{Organization: projectOrganization(res)}, nil
}

// Organizations delegates to service.Organizations.
func (h *AdminHandler) Organizations(ctx context.Context, req *authorizerv1.OrganizationsRequest) (*authorizerv1.OrganizationsResponse, error) {
	res, _, err := h.Service.Organizations(ctx, transport.MetaFromGRPC(ctx), &model.ListOrganizationsRequest{
		Pagination: modelPaginationRequest(req.GetPagination()),
	})
	if err != nil {
		return nil, err
	}
	return projectOrganizations(res), nil
}

// UserOrganizations delegates to service.UserOrganizations.
func (h *AdminHandler) UserOrganizations(ctx context.Context, req *authorizerv1.UserOrganizationsRequest) (*authorizerv1.UserOrganizationsResponse, error) {
	res, _, err := h.Service.UserOrganizations(ctx, transport.MetaFromGRPC(ctx), &model.UserOrganizationsRequest{
		UserID:     req.GetUserId(),
		Pagination: modelPaginationRequest(req.GetPagination()),
	})
	if err != nil {
		return nil, err
	}
	return projectUserOrganizations(res), nil
}

// AddOrgMember delegates to service.AddOrgMember. An omitted roles list arrives
// as nil, which the service treats as the empty set.
func (h *AdminHandler) AddOrgMember(ctx context.Context, req *authorizerv1.AddOrgMemberRequest) (*authorizerv1.AddOrgMemberResponse, error) {
	res, _, err := h.Service.AddOrgMember(ctx, transport.MetaFromGRPC(ctx), &model.AddOrgMemberRequest{
		OrgID:  req.GetOrgId(),
		UserID: req.GetUserId(),
		Roles:  req.GetRoles(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.AddOrgMemberResponse{OrgMember: projectOrgMember(res)}, nil
}

// RemoveOrgMember delegates to service.RemoveOrgMember.
func (h *AdminHandler) RemoveOrgMember(ctx context.Context, req *authorizerv1.RemoveOrgMemberRequest) (*authorizerv1.RemoveOrgMemberResponse, error) {
	res, _, err := h.Service.RemoveOrgMember(ctx, transport.MetaFromGRPC(ctx), &model.RemoveOrgMemberRequest{
		OrgID:  req.GetOrgId(),
		UserID: req.GetUserId(),
	})
	if err != nil {
		return nil, err
	}
	return &authorizerv1.RemoveOrgMemberResponse{Message: res.Message}, nil
}

// OrgMembers delegates to service.OrgMembers.
func (h *AdminHandler) OrgMembers(ctx context.Context, req *authorizerv1.OrgMembersRequest) (*authorizerv1.OrgMembersResponse, error) {
	res, _, err := h.Service.OrgMembers(ctx, transport.MetaFromGRPC(ctx), &model.ListOrgMembersRequest{
		OrgID:      req.GetOrgId(),
		Pagination: modelPaginationRequest(req.GetPagination()),
	})
	if err != nil {
		return nil, err
	}
	return projectOrgMembers(res), nil
}

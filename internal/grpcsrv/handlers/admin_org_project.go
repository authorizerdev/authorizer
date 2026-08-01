package handlers

import (
	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Projectors for the organization / membership admin surface. Each mirrors the
// GraphQL model 1:1; optional model pointers map onto proto `optional` fields so
// "unset" survives the hop instead of collapsing to a zero value.

func projectOrganization(o *model.Organization) *authorizerv1.Organization {
	if o == nil {
		return nil
	}
	return &authorizerv1.Organization{
		Id:          o.ID,
		Name:        o.Name,
		DisplayName: o.DisplayName,
		Enabled:     o.Enabled,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}
}

func projectOrgMember(m *model.OrgMember) *authorizerv1.OrgMember {
	if m == nil {
		return nil
	}
	return &authorizerv1.OrgMember{
		Id:         m.ID,
		OrgId:      m.OrgID,
		UserId:     m.UserID,
		Email:      m.Email,
		GivenName:  m.GivenName,
		FamilyName: m.FamilyName,
		Roles:      m.Roles,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func projectUserOrganization(u *model.UserOrganization) *authorizerv1.UserOrganization {
	if u == nil {
		return nil
	}
	return &authorizerv1.UserOrganization{
		Organization: projectOrganization(u.Organization),
		Roles:        u.Roles,
	}
}

func projectOrganizations(o *model.Organizations) *authorizerv1.OrganizationsResponse {
	if o == nil {
		return &authorizerv1.OrganizationsResponse{}
	}
	items := make([]*authorizerv1.Organization, 0, len(o.Organizations))
	for _, item := range o.Organizations {
		items = append(items, projectOrganization(item))
	}
	return &authorizerv1.OrganizationsResponse{
		Organizations: items,
		Pagination:    projectPagination(o.Pagination),
	}
}

func projectOrgMembers(m *model.OrgMembers) *authorizerv1.OrgMembersResponse {
	if m == nil {
		return &authorizerv1.OrgMembersResponse{}
	}
	items := make([]*authorizerv1.OrgMember, 0, len(m.OrgMembers))
	for _, item := range m.OrgMembers {
		items = append(items, projectOrgMember(item))
	}
	return &authorizerv1.OrgMembersResponse{
		OrgMembers: items,
		Pagination: projectPagination(m.Pagination),
	}
}

func projectUserOrganizations(u *model.UserOrganizations) *authorizerv1.UserOrganizationsResponse {
	if u == nil {
		return &authorizerv1.UserOrganizationsResponse{}
	}
	items := make([]*authorizerv1.UserOrganization, 0, len(u.UserOrganizations))
	for _, item := range u.UserOrganizations {
		items = append(items, projectUserOrganization(item))
	}
	return &authorizerv1.UserOrganizationsResponse{
		UserOrganizations: items,
		Pagination:        projectPagination(u.Pagination),
	}
}

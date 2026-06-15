// Package handlers — admin_project.go holds proto<->model projection helpers
// for the admin surface, mirroring project.go for the public surface. Each
// helper converts a GraphQL/storage model type returned by service.* into the
// proto wire type so the admin handler methods stay focused on delegation.
package handlers

import (
	"github.com/authorizerdev/authorizer/internal/graph/model"

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

package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AdminMeta returns admin-only configuration metadata — the configured roles,
// default roles and protected roles. It is the non-deprecated replacement for
// the bits of _env the dashboard needs (e.g. seeding the FGA model builder with
// the instance's real roles and flagging FGA role references that aren't
// configured roles).
// Permissions: authorizer:admin
func (g *graphqlProvider) AdminMeta(ctx context.Context) (*model.AdminMeta, error) {
	log := g.Log.With().Str("func", "AdminMeta").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}
	// Never return nil slices: the schema fields are non-null lists, so default
	// to empty slices when nothing is configured.
	roles := g.Config.Roles
	if roles == nil {
		roles = []string{}
	}
	defaultRoles := g.Config.DefaultRoles
	if defaultRoles == nil {
		defaultRoles = []string{}
	}
	protectedRoles := g.Config.ProtectedRoles
	if protectedRoles == nil {
		protectedRoles = []string{}
	}
	return &model.AdminMeta{
		Roles:          roles,
		DefaultRoles:   defaultRoles,
		ProtectedRoles: protectedRoles,
	}, nil
}

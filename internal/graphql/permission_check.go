package graphql

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/authorization"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// enforceRequiredPermissions evaluates each required permission against the
// authorization provider with AND semantics — every entry must be allowed,
// otherwise the caller is treated as unauthorized.
//
// When required is empty (the common case) this is a no-op and existing
// callers see no behavior change.
func (g *graphqlProvider) enforceRequiredPermissions(
	ctx context.Context,
	log zerolog.Logger,
	userID string,
	roles []string,
	required []*model.PermissionInput,
) error {
	if len(required) == 0 {
		return nil
	}
	principal := &authorization.Principal{
		ID:    userID,
		Type:  constants.PrincipalTypeUser,
		Roles: roles,
	}
	for _, p := range required {
		if p == nil {
			continue
		}
		res, err := g.AuthorizationProvider.CheckPermission(ctx, principal, p.Resource, p.Scope)
		if err != nil {
			log.Debug().Err(err).Str("resource", p.Resource).Str("scope", p.Scope).Msg("required permission check errored")
			return errors.New("unauthorized")
		}
		if res == nil || !res.Allowed {
			log.Debug().Str("resource", p.Resource).Str("scope", p.Scope).Msg("required permission denied")
			return errors.New("unauthorized")
		}
	}
	return nil
}

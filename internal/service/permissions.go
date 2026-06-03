package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/authorization"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Permissions returns every (resource, scope) pair the authenticated user
// is allowed to act on, derived from their roles and the policy engine.
// Transport-agnostic port of graphqlProvider.Permissions.
//
// Permissions: authenticated user.
func (p *provider) Permissions(ctx context.Context, meta RequestMetadata) ([]*model.Permission, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Permissions").Logger()

	gc := &gin.Context{Request: meta.Request}
	tokenData, err := p.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user from token")
		return nil, nil, fmt.Errorf("unauthorized")
	}

	user, err := p.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by ID")
		return nil, nil, err
	}

	var roles []string
	if user.Roles != "" {
		roles = strings.Split(user.Roles, ",")
	}

	principal := &authorization.Principal{
		ID:    user.ID,
		Type:  constants.PrincipalTypeUser,
		Roles: roles,
	}

	resourceScopes, err := p.AuthorizationProvider.GetPrincipalPermissions(ctx, principal)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get principal permissions")
		return nil, nil, err
	}

	res := make([]*model.Permission, len(resourceScopes))
	for i, rs := range resourceScopes {
		res[i] = &model.Permission{
			Resource: rs.Resource,
			Scope:    rs.Scope,
		}
	}
	return res, nil, nil
}

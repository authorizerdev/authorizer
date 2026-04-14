package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/authorization"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// MyPermissions is the method to get all permissions for the authenticated user.
// Permissions: authorized user
func (g *graphqlProvider) MyPermissions(ctx context.Context) ([]*model.AuthzResourceScope, error) {
	log := g.Log.With().Str("func", "MyPermissions").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	tokenData, err := g.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user from token")
		return nil, fmt.Errorf("unauthorized")
	}

	user, err := g.StorageProvider.GetUserByID(ctx, tokenData.UserID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user by ID")
		return nil, err
	}

	var roles []string
	if user.Roles != "" {
		roles = strings.Split(user.Roles, ",")
	}

	principal := &authorization.Principal{
		ID:    user.ID,
		Type:  "user",
		Roles: roles,
	}

	resourceScopes, err := g.AuthorizationProvider.GetPrincipalPermissions(ctx, principal)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get principal permissions")
		return nil, err
	}

	res := make([]*model.AuthzResourceScope, len(resourceScopes))
	for i, rs := range resourceScopes {
		res[i] = &model.AuthzResourceScope{
			Resource: rs.Resource,
			Scope:    rs.Scope,
		}
	}

	return res, nil
}

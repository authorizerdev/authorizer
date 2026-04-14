package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/authorization"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// CheckPermission is the method to check if the authenticated user has a specific permission.
// Permissions: authorized user
func (g *graphqlProvider) CheckPermission(ctx context.Context, params *model.CheckPermissionInput) (*model.CheckPermissionResponse, error) {
	log := g.Log.With().Str("func", "CheckPermission").Logger()
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

	result, err := g.AuthorizationProvider.CheckPermission(ctx, principal, params.Resource, params.Scope)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to check permission")
		return nil, err
	}

	resp := &model.CheckPermissionResponse{
		Allowed: result.Allowed,
	}
	if result.MatchedPolicy != "" {
		resp.MatchedPolicy = &result.MatchedPolicy
	}

	return resp, nil
}

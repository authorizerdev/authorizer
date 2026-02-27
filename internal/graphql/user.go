package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// User is the method to get user details
// Permission: authorizer:admin
func (g *graphqlProvider) User(ctx context.Context, params *model.GetUserRequest) (*model.User, error) {
	log := g.Log.With().Str("func", "User").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}
	// Try getting user by ID
	if params.ID != nil && strings.Trim(*params.ID, " ") != "" {
		res, err := g.StorageProvider.GetUserByID(ctx, *params.ID)
		if err != nil {
			log.Debug().Err(err).Msg("failed GetUserByID")
			return nil, err
		}
		return res.AsAPIUser(), nil
	}
	// Try getting user by email
	if params.Email != nil && strings.Trim(*params.Email, " ") != "" {
		res, err := g.StorageProvider.GetUserByEmail(ctx, *params.Email)
		if err != nil {
			log.Debug().Err(err).Msg("failed GetUserByEmail")
			return nil, err
		}
		return res.AsAPIUser(), nil
	}
	// Return error if no params are provided
	return nil, fmt.Errorf("invalid params, user id or email is required")
}

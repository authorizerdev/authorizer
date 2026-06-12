package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// FgaListUsers returns the fully-qualified user ids of user_type that have
// relation on object ("who can access this object?"). This is an introspection
// surface that reveals the access graph, so it is super-admin gated like the
// other _fga_* admin ops. Read-only: no audit.
// Permission: authorizer:admin.
func (g *graphqlProvider) FgaListUsers(ctx context.Context, params *model.FgaListUsersInput) (*model.FgaListUsersResponse, error) {
	log := g.Log.With().Str("func", "FgaListUsers").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}
	if !g.TokenProvider.IsSuperAdmin(gc) {
		log.Debug().Msg("Not logged in as super admin")
		return nil, fmt.Errorf("unauthorized")
	}
	if g.AuthzEngine == nil {
		return nil, errFgaNotEnabled
	}
	if params == nil || strings.TrimSpace(params.Object) == "" || strings.TrimSpace(params.Relation) == "" || strings.TrimSpace(params.UserType) == "" {
		return nil, fmt.Errorf("object, relation and user_type are required")
	}
	users, err := g.AuthzEngine.ListUsers(ctx, params.Object, params.Relation, params.UserType)
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpListUsers, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Failed to list users")
		return nil, err
	}
	metrics.RecordFgaOperation(metrics.FgaOpListUsers, metrics.FgaResultSuccess)
	// Cap the result set; ListUsers is an expensive enumeration surface.
	if len(users) > maxFgaListResults {
		users = users[:maxFgaListResults]
	}
	return &model.FgaListUsersResponse{Users: users}, nil
}

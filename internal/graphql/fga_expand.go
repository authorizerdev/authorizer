package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// FgaExpand returns the OpenFGA relationship/userset tree for (relation, object)
// as a JSON string (the explainability/"why" primitive). It reveals the access
// graph, so it is super-admin gated like the other _fga_* admin ops. Read-only:
// no audit.
// Permission: authorizer:admin.
func (g *graphqlProvider) FgaExpand(ctx context.Context, params *model.FgaExpandInput) (*model.FgaExpandResponse, error) {
	log := g.Log.With().Str("func", "FgaExpand").Logger()
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
	if params == nil || strings.TrimSpace(params.Relation) == "" || strings.TrimSpace(params.Object) == "" {
		return nil, fmt.Errorf("relation and object are required")
	}
	tree, err := g.AuthzEngine.Expand(ctx, params.Relation, params.Object)
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpExpand, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Failed to expand")
		return nil, err
	}
	metrics.RecordFgaOperation(metrics.FgaOpExpand, metrics.FgaResultSuccess)
	return &model.FgaExpandResponse{Tree: tree}, nil
}

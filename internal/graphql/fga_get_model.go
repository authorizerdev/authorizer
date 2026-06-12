package graphql

import (
	"context"
	"errors"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// FgaGetModel returns the active fine-grained authorization model as DSL.
// Permission: authorizer:admin.
func (g *graphqlProvider) FgaGetModel(ctx context.Context) (*model.FgaModel, error) {
	log := g.Log.With().Str("func", "FgaGetModel").Logger()
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
	id, dsl, err := g.AuthzEngine.ReadModel(ctx)
	if err != nil {
		// A store with no model yet is a normal empty state, not a failure:
		// return an empty model so the dashboard shows its "define a model"
		// starting point instead of an error.
		if errors.Is(err, engine.ErrNoModel) {
			metrics.RecordFgaOperation(metrics.FgaOpGetModel, metrics.FgaResultSuccess)
			return &model.FgaModel{ID: "", Dsl: ""}, nil
		}
		metrics.RecordFgaOperation(metrics.FgaOpGetModel, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Failed to read authorization model")
		return nil, err
	}
	metrics.RecordFgaOperation(metrics.FgaOpGetModel, metrics.FgaResultSuccess)
	return &model.FgaModel{ID: id, Dsl: dsl}, nil
}

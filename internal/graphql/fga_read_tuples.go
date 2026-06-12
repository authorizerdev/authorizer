package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// FgaReadTuples returns a page of persisted tuples matching the filter.
// Permission: authorizer:admin.
func (g *graphqlProvider) FgaReadTuples(ctx context.Context, params *model.FgaReadTuplesInput) (*model.FgaTuples, error) {
	log := g.Log.With().Str("func", "FgaReadTuples").Logger()
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
	filter := engine.ReadTuplesFilter{}
	if params != nil {
		filter.User = refs.StringValue(params.User)
		filter.Relation = refs.StringValue(params.Relation)
		filter.Object = refs.StringValue(params.Object)
		filter.ContinuationToken = refs.StringValue(params.ContinuationToken)
		if params.PageSize != nil {
			ps := *params.PageSize
			// Cap page size; it is an enumeration surface and the backend
			// enforces a [1, 100] range.
			if ps <= 0 || ps > maxFgaReadPageSize {
				ps = maxFgaReadPageSize
			}
			filter.PageSize = int32(ps)
		}
	}
	if filter.PageSize == 0 {
		filter.PageSize = maxFgaReadPageSize
	}
	res, err := g.AuthzEngine.ReadTuples(ctx, filter)
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpReadTuples, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Failed to read tuples")
		return nil, err
	}
	metrics.RecordFgaOperation(metrics.FgaOpReadTuples, metrics.FgaResultSuccess)
	out := &model.FgaTuples{Tuples: make([]*model.FgaTuple, 0, len(res.Tuples))}
	for _, t := range res.Tuples {
		out.Tuples = append(out.Tuples, &model.FgaTuple{User: t.User, Relation: t.Relation, Object: t.Object})
	}
	if res.ContinuationToken != "" {
		out.ContinuationToken = refs.NewStringRef(res.ContinuationToken)
	}
	return out, nil
}

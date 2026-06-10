package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// FgaDeleteTuples removes the given relationship tuples.
// Permission: authorizer:admin. Audited.
func (g *graphqlProvider) FgaDeleteTuples(ctx context.Context, params *model.FgaWriteTuplesInput) (*model.Response, error) {
	log := g.Log.With().Str("func", "FgaDeleteTuples").Logger()
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
	tuples, err := toEngineTuples(params)
	if err != nil {
		return nil, err
	}
	if err := g.AuthzEngine.DeleteTuples(ctx, tuples); err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpDeleteTuples, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Failed to delete tuples")
		return nil, err
	}
	metrics.RecordFgaOperation(metrics.FgaOpDeleteTuples, metrics.FgaResultSuccess)
	g.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminFgaTuplesDeletedEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeFgaTuple,
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
		Metadata:     fmt.Sprintf("count=%d", len(tuples)),
	})
	return &model.Response{Message: "Tuples deleted successfully"}, nil
}

package graphql

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// FgaReset deletes the entire fine-grained authorization store — the model, all
// its versions, and all tuples — and starts a fresh, empty store. OpenFGA has no
// per-version model delete, so this is the only way to remove a model.
//
// It is guarded: the reset is refused while any relationship tuples still exist,
// so an admin cannot silently drop live grants. Callers must delete all tuples
// first. Destructive and audited.
// Permission: authorizer:admin.
func (g *graphqlProvider) FgaReset(ctx context.Context) (*model.Response, error) {
	log := g.Log.With().Str("func", "FgaReset").Logger()
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
	// Safety gate: refuse to reset while tuples still exist so live grants are
	// never dropped without an explicit prior cleanup.
	existing, err := g.AuthzEngine.ReadTuples(ctx, engine.ReadTuplesFilter{PageSize: 1})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to check for existing tuples before reset")
		return nil, err
	}
	if existing != nil && len(existing.Tuples) > 0 {
		return nil, fmt.Errorf("remove all relationship tuples before resetting the authorization model")
	}
	if err := g.AuthzEngine.Reset(ctx); err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpReset, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Failed to reset authorization store")
		return nil, err
	}
	metrics.RecordFgaOperation(metrics.FgaOpReset, metrics.FgaResultSuccess)
	g.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminFgaResetEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeFgaModel,
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
	})
	return &model.Response{Message: "Authorization model reset successfully"}, nil
}

package graphql

import (
	"context"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// FgaWriteModel installs a new fine-grained authorization model from its DSL.
// Permission: authorizer:admin. Audited.
func (g *graphqlProvider) FgaWriteModel(ctx context.Context, params *model.FgaWriteModelInput) (*model.FgaModel, error) {
	log := g.Log.With().Str("func", "FgaWriteModel").Logger()
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
	if params == nil || strings.TrimSpace(params.Dsl) == "" {
		return nil, fmt.Errorf("dsl is required")
	}
	modelID, err := g.AuthzEngine.WriteModel(ctx, params.Dsl)
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpWriteModel, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Failed to write authorization model")
		return nil, err
	}
	metrics.RecordFgaOperation(metrics.FgaOpWriteModel, metrics.FgaResultSuccess)
	g.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminFgaModelWrittenEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeFgaModel,
		ResourceID:   modelID,
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
	})
	return &model.FgaModel{ID: modelID, Dsl: params.Dsl}, nil
}

package graphql

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// errFgaNotEnabled is returned by every FGA resolver when no authorization
// engine is configured (no --fga-store). Fail-closed.
var errFgaNotEnabled = errors.New("fine-grained authorization is not enabled")

// maxFgaTuplesPerWrite caps the number of tuples accepted in a single write or
// delete to bound the work an admin call performs.
const maxFgaTuplesPerWrite = 100

// maxFgaReadPageSize caps the page size for tuple reads. OpenFGA's ReadRequest
// enforces a [1, 100] range, so this is both a safety cap and a hard backend
// limit.
const maxFgaReadPageSize = 100

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
		log.Debug().Err(err).Msg("Failed to write authorization model")
		return nil, err
	}
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
		log.Debug().Err(err).Msg("Failed to read authorization model")
		return nil, err
	}
	return &model.FgaModel{ID: id, Dsl: dsl}, nil
}

// FgaWriteTuples persists the given relationship tuples.
// Permission: authorizer:admin. Audited.
func (g *graphqlProvider) FgaWriteTuples(ctx context.Context, params *model.FgaWriteTuplesInput) (*model.Response, error) {
	log := g.Log.With().Str("func", "FgaWriteTuples").Logger()
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
	if err := g.AuthzEngine.WriteTuples(ctx, tuples); err != nil {
		log.Debug().Err(err).Msg("Failed to write tuples")
		return nil, err
	}
	g.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminFgaTuplesWrittenEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeFgaTuple,
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
		Metadata:     fmt.Sprintf("count=%d", len(tuples)),
	})
	return &model.Response{Message: "Tuples written successfully"}, nil
}

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
		log.Debug().Err(err).Msg("Failed to delete tuples")
		return nil, err
	}
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
		log.Debug().Err(err).Msg("Failed to read tuples")
		return nil, err
	}
	out := &model.FgaTuples{Tuples: make([]*model.FgaTuple, 0, len(res.Tuples))}
	for _, t := range res.Tuples {
		out.Tuples = append(out.Tuples, &model.FgaTuple{User: t.User, Relation: t.Relation, Object: t.Object})
	}
	if res.ContinuationToken != "" {
		out.ContinuationToken = refs.NewStringRef(res.ContinuationToken)
	}
	return out, nil
}

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
		log.Debug().Err(err).Msg("Failed to list users")
		return nil, err
	}
	// Cap the result set; ListUsers is an expensive enumeration surface.
	if len(users) > maxFgaListResults {
		users = users[:maxFgaListResults]
	}
	return &model.FgaListUsersResponse{Users: users}, nil
}

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
		log.Debug().Err(err).Msg("Failed to expand")
		return nil, err
	}
	return &model.FgaExpandResponse{Tree: tree}, nil
}

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
		log.Debug().Err(err).Msg("Failed to reset authorization store")
		return nil, err
	}
	g.AuditProvider.LogEvent(audit.Event{
		Action:       constants.AuditAdminFgaResetEvent,
		ActorType:    constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeFgaModel,
		IPAddress:    utils.GetIP(gc.Request),
		UserAgent:    utils.GetUserAgent(gc.Request),
	})
	return &model.Response{Message: "Authorization model reset successfully"}, nil
}

// toEngineTuples validates and converts admin-supplied tuple inputs into engine
// tuples. It enforces a per-call cap and rejects empty fields.
func toEngineTuples(params *model.FgaWriteTuplesInput) ([]engine.TupleKey, error) {
	if params == nil || len(params.Tuples) == 0 {
		return nil, fmt.Errorf("at least one tuple is required")
	}
	if len(params.Tuples) > maxFgaTuplesPerWrite {
		return nil, fmt.Errorf("too many tuples: max %d per request", maxFgaTuplesPerWrite)
	}
	tuples := make([]engine.TupleKey, 0, len(params.Tuples))
	for _, t := range params.Tuples {
		if t == nil || strings.TrimSpace(t.User) == "" || strings.TrimSpace(t.Relation) == "" || strings.TrimSpace(t.Object) == "" {
			return nil, fmt.Errorf("each tuple requires user, relation and object")
		}
		tuples = append(tuples, engine.TupleKey{User: t.User, Relation: t.Relation, Object: t.Object})
	}
	return tuples, nil
}

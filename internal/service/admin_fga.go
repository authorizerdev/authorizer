package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// maxFgaTuplesPerWrite caps the number of tuples accepted in a single write or
// delete to bound the work an admin call performs.
const maxFgaTuplesPerWrite = 100

// maxFgaConditionParams caps the parameters stored with one conditional
// tuple. Conditions declare a handful of typed parameters; anything larger is
// a malformed or abusive payload.
const maxFgaConditionParams = 32

// maxFgaReadPageSize caps the page size for tuple reads. OpenFGA's ReadRequest
// enforces a [1, 100] range, so this is both a safety cap and a hard backend
// limit.
const maxFgaReadPageSize = 100

// FgaGetModel returns the active fine-grained authorization model as DSL. A
// store with no model yet is a normal empty state (returns an empty model, not
// an error) so the dashboard can show its "define a model" starting point.
// Requires super-admin auth. Fail-closed: a nil engine returns
// ErrFgaNotEnabled. Logic migrated from internal/graphql/fga_get_model.go.
// Permission: authorizer:admin.
func (p *provider) FgaGetModel(ctx context.Context, meta RequestMetadata) (*model.FgaModel, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "FgaGetModel").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	if p.AuthzEngine == nil {
		return nil, nil, ErrFgaNotEnabled
	}
	id, dsl, err := p.AuthzEngine.ReadModel(ctx)
	if err != nil {
		if errors.Is(err, engine.ErrNoModel) {
			metrics.RecordFgaOperation(metrics.FgaOpGetModel, metrics.FgaResultSuccess)
			return &model.FgaModel{ID: "", Dsl: ""}, nil, nil
		}
		metrics.RecordFgaOperation(metrics.FgaOpGetModel, metrics.FgaResultError)
		log.Debug().Err(err).Msg("Failed to read authorization model")
		return nil, nil, err
	}
	metrics.RecordFgaOperation(metrics.FgaOpGetModel, metrics.FgaResultSuccess)
	return &model.FgaModel{ID: id, Dsl: dsl}, nil, nil
}

// FgaWriteModel installs a new fine-grained authorization model from its DSL and
// returns the new model id. Requires super-admin auth. Fail-closed: a nil engine
// returns ErrFgaNotEnabled. Audited. Logic migrated from
// internal/graphql/fga_write_model.go.
// Permission: authorizer:admin.
func (p *provider) FgaWriteModel(ctx context.Context, meta RequestMetadata, params *model.FgaWriteModelInput) (*model.FgaModel, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "FgaWriteModel").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	if p.AuthzEngine == nil {
		return nil, nil, ErrFgaNotEnabled
	}
	if params == nil || strings.TrimSpace(params.Dsl) == "" {
		return nil, nil, InvalidArgument("dsl is required")
	}
	// Outcome metric is recorded by the engine (WriteModel) itself.
	modelID, err := p.AuthzEngine.WriteModel(ctx, params.Dsl)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to write authorization model")
		return nil, nil, err
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminFgaModelWrittenEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeFgaModel,
		ResourceID:   modelID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.FgaModel{ID: modelID, Dsl: params.Dsl}, nil, nil
}

// FgaWriteTuples persists the given relationship tuples (additive) and returns a
// status message. Requires super-admin auth. Fail-closed: a nil engine returns
// ErrFgaNotEnabled. Audited. Logic migrated from
// internal/graphql/fga_write_tuples.go.
// Permission: authorizer:admin.
func (p *provider) FgaWriteTuples(ctx context.Context, meta RequestMetadata, params *model.FgaWriteTuplesInput) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "FgaWriteTuples").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	if p.AuthzEngine == nil {
		return nil, nil, ErrFgaNotEnabled
	}
	tuples, err := toEngineTuples(params, true)
	if err != nil {
		return nil, nil, err
	}
	if err := p.requireBoundConditions(ctx, tuples); err != nil {
		return nil, nil, err
	}
	// Outcome metric is recorded by the engine (WriteTuples) itself.
	if err := p.AuthzEngine.WriteTuples(ctx, tuples); err != nil {
		log.Debug().Err(err).Msg("Failed to write tuples")
		return nil, nil, friendlyTupleError(err)
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminFgaTuplesWrittenEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeFgaTuple,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
		Metadata:     fmt.Sprintf("count=%d", len(tuples)),
	})
	return &model.Response{Message: "Tuples written successfully"}, nil, nil
}

// FgaDeleteTuples removes the given relationship tuples and returns a status
// message. Requires super-admin auth. Fail-closed: a nil engine returns
// ErrFgaNotEnabled. Audited. Logic migrated from
// internal/graphql/fga_delete_tuples.go.
// Permission: authorizer:admin.
func (p *provider) FgaDeleteTuples(ctx context.Context, meta RequestMetadata, params *model.FgaWriteTuplesInput) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "FgaDeleteTuples").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	if p.AuthzEngine == nil {
		return nil, nil, ErrFgaNotEnabled
	}
	tuples, err := toEngineTuples(params, false)
	if err != nil {
		return nil, nil, err
	}
	// Outcome metric is recorded by the engine (DeleteTuples) itself.
	if err := p.AuthzEngine.DeleteTuples(ctx, tuples); err != nil {
		log.Debug().Err(err).Msg("Failed to delete tuples")
		return nil, nil, friendlyTupleError(err)
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminFgaTuplesDeletedEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeFgaTuple,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
		Metadata:     fmt.Sprintf("count=%d", len(tuples)),
	})
	return &model.Response{Message: "Tuples deleted successfully"}, nil, nil
}

// FgaReadTuples returns a page of persisted tuples matching the filter. Requires
// super-admin auth. Fail-closed: a nil engine returns ErrFgaNotEnabled.
// Read-only: no audit. Logic migrated from internal/graphql/fga_read_tuples.go.
// Permission: authorizer:admin.
func (p *provider) FgaReadTuples(ctx context.Context, meta RequestMetadata, params *model.FgaReadTuplesInput) (*model.FgaTuples, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "FgaReadTuples").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	if p.AuthzEngine == nil {
		return nil, nil, ErrFgaNotEnabled
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
	// Outcome metric is recorded by the engine (ReadTuples) itself.
	res, err := p.AuthzEngine.ReadTuples(ctx, filter)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read tuples")
		return nil, nil, err
	}
	out := &model.FgaTuples{Tuples: make([]*model.FgaTuple, 0, len(res.Tuples))}
	for _, t := range res.Tuples {
		ft := &model.FgaTuple{User: t.User, Relation: t.Relation, Object: t.Object}
		if t.Condition != nil {
			ft.Condition = &model.FgaCondition{Name: t.Condition.Name, Context: t.Condition.Context}
		}
		out.Tuples = append(out.Tuples, ft)
	}
	if res.ContinuationToken != "" {
		out.ContinuationToken = refs.NewStringRef(res.ContinuationToken)
	}
	return out, nil, nil
}

// FgaListUsers returns the fully-qualified user ids of user_type that have
// relation on object ("who can access this object?"). This is an introspection
// surface that reveals the access graph, so it is super-admin gated. Requires
// super-admin auth. Fail-closed: a nil engine returns ErrFgaNotEnabled.
// Read-only: no audit. Logic migrated from internal/graphql/fga_list_users.go.
// Permission: authorizer:admin.
func (p *provider) FgaListUsers(ctx context.Context, meta RequestMetadata, params *model.FgaListUsersInput) (*model.FgaListUsersResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "FgaListUsers").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	if p.AuthzEngine == nil {
		return nil, nil, ErrFgaNotEnabled
	}
	if params == nil || strings.TrimSpace(params.Object) == "" || strings.TrimSpace(params.Relation) == "" || strings.TrimSpace(params.UserType) == "" {
		return nil, nil, InvalidArgument("object, relation and user_type are required")
	}
	// Outcome metric is recorded by the engine (ListUsers) itself.
	users, err := p.AuthzEngine.ListUsers(ctx, params.Object, params.Relation, params.UserType)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to list users")
		return nil, nil, err
	}
	// Cap the result set; ListUsers is an expensive enumeration surface.
	if len(users) > maxFgaListResults {
		users = users[:maxFgaListResults]
	}
	return &model.FgaListUsersResponse{Users: users}, nil, nil
}

// FgaExpand returns the OpenFGA relationship/userset tree for (relation, object)
// as a JSON string (the explainability/"why" primitive). It reveals the access
// graph, so it is super-admin gated. Requires super-admin auth. Fail-closed: a
// nil engine returns ErrFgaNotEnabled. Read-only: no audit. Logic migrated from
// internal/graphql/fga_expand.go.
// Permission: authorizer:admin.
func (p *provider) FgaExpand(ctx context.Context, meta RequestMetadata, params *model.FgaExpandInput) (*model.FgaExpandResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "FgaExpand").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	if p.AuthzEngine == nil {
		return nil, nil, ErrFgaNotEnabled
	}
	if params == nil || strings.TrimSpace(params.Relation) == "" || strings.TrimSpace(params.Object) == "" {
		return nil, nil, InvalidArgument("relation and object are required")
	}
	// Outcome metric is recorded by the engine (Expand) itself.
	tree, err := p.AuthzEngine.Expand(ctx, params.Relation, params.Object)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to expand")
		return nil, nil, err
	}
	return &model.FgaExpandResponse{Tree: tree}, nil, nil
}

// FgaReset deletes the entire fine-grained authorization store — the model, all
// its versions, and all tuples — and starts a fresh, empty store. OpenFGA has no
// per-version model delete, so this is the only way to remove a model.
//
// It is guarded: the reset is refused while any relationship tuples still exist,
// so an admin cannot silently drop live grants. Callers must delete all tuples
// first. Requires super-admin auth. Fail-closed: a nil engine returns
// ErrFgaNotEnabled. Destructive and audited. Logic migrated from
// internal/graphql/fga_reset.go.
// Permission: authorizer:admin.
func (p *provider) FgaReset(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "FgaReset").Logger()
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	if p.AuthzEngine == nil {
		return nil, nil, ErrFgaNotEnabled
	}
	// Safety gate: refuse to reset while tuples still exist so live grants are
	// never dropped without an explicit prior cleanup.
	existing, err := p.AuthzEngine.ReadTuples(ctx, engine.ReadTuplesFilter{PageSize: 1})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to check for existing tuples before reset")
		return nil, nil, err
	}
	if existing != nil && len(existing.Tuples) > 0 {
		return nil, nil, FailedPrecondition("remove all relationship tuples before resetting the authorization model")
	}
	// Outcome metric is recorded by the engine (Reset) itself.
	if err := p.AuthzEngine.Reset(ctx); err != nil {
		log.Debug().Err(err).Msg("Failed to reset authorization store")
		return nil, nil, err
	}
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminFgaResetEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeFgaModel,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.Response{Message: "Authorization model reset successfully"}, nil, nil
}

// toEngineTuples validates and converts admin-supplied tuple inputs into engine
// tuples. It enforces a per-call cap and rejects empty fields.
// toEngineTuples validates and converts admin tuple input. allowCondition is
// true for writes; a delete matches on (user, relation, object) alone, so a
// condition there is rejected rather than silently ignored.
func toEngineTuples(params *model.FgaWriteTuplesInput, allowCondition bool) ([]engine.TupleKey, error) {
	if params == nil || len(params.Tuples) == 0 {
		return nil, InvalidArgument("at least one tuple is required")
	}
	if len(params.Tuples) > maxFgaTuplesPerWrite {
		return nil, InvalidArgument(fmt.Sprintf("too many tuples: max %d per request", maxFgaTuplesPerWrite))
	}
	tuples := make([]engine.TupleKey, 0, len(params.Tuples))
	for _, t := range params.Tuples {
		if t == nil || strings.TrimSpace(t.User) == "" || strings.TrimSpace(t.Relation) == "" || strings.TrimSpace(t.Object) == "" {
			return nil, InvalidArgument("each tuple requires user, relation and object")
		}
		key := engine.TupleKey{User: t.User, Relation: t.Relation, Object: t.Object}
		if t.Condition != nil {
			if !allowCondition {
				return nil, InvalidArgument("condition is not used when deleting tuples; delete by user, relation and object")
			}
			cond, err := toEngineCondition(t.Condition)
			if err != nil {
				return nil, err
			}
			key.Condition = cond
		}
		tuples = append(tuples, key)
	}
	return tuples, nil
}

// toEngineCondition validates a tuple condition. The condition name and its
// parameter types are checked against the model by the engine on write.
func toEngineCondition(c *model.FgaConditionInput) (*engine.TupleCondition, error) {
	name := strings.TrimSpace(c.Name)
	if name == "" {
		return nil, InvalidArgument("condition requires a name declared in the authorization model")
	}
	if len(c.Context) > maxFgaConditionParams {
		return nil, InvalidArgument(fmt.Sprintf("too many condition parameters: max %d", maxFgaConditionParams))
	}
	if _, ok := c.Context[engine.ContextKeyCurrentTime]; ok {
		return nil, InvalidArgument("current_time cannot be stored on a tuple; the server supplies it at check time")
	}
	return &engine.TupleCondition{Name: name, Context: c.Context}, nil
}

// tupleValidationRe extracts the useful part of OpenFGA's tuple-validation
// error (e.g. `Invalid tuple 'document:9#owner@user:abc'. Reason: relation
// 'document#owner' not found`) from the raw gRPC error string.
var tupleValidationRe = regexp.MustCompile(`Invalid tuple '([^']+)'\. Reason: (.+)$`)

// friendlyTupleError turns OpenFGA's raw tuple-validation gRPC error into an
// actionable message ("relation X not found — define it in the model first").
// Non-validation errors pass through unchanged; the raw error stays in the
// debug log at the call site.
func friendlyTupleError(err error) error {
	m := tupleValidationRe.FindStringSubmatch(err.Error())
	if m == nil {
		return err
	}
	return InvalidArgument(fmt.Sprintf("invalid tuple %q: %s — the relation and object type must be defined in the active authorization model (Step 1)", m[1], m[2]))
}

// requireBoundConditions rejects a conditional tuple that leaves any condition
// parameter other than current_time unbound.
//
// Most evaluation surfaces have no way to supply request context: list
// permissions, required_relations, the SAML group assertion and org-derived
// roles evaluate with the server's current_time alone. A tuple whose condition
// still needs another request-time parameter makes OpenFGA error on every one
// of those paths, failing closed and silently emptying, for example, the
// user's SAML groups. Requiring every parameter at write time keeps every
// stored condition evaluable everywhere.
func (p *provider) requireBoundConditions(ctx context.Context, tuples []engine.TupleKey) error {
	var declared map[string][]string
	for _, t := range tuples {
		if t.Condition == nil {
			continue
		}
		if declared == nil {
			d, err := p.AuthzEngine.ConditionParams(ctx)
			if err != nil {
				return err
			}
			declared = d
		}
		params, ok := declared[t.Condition.Name]
		if !ok {
			return InvalidArgument(fmt.Sprintf("condition %q is not declared in the active authorization model", t.Condition.Name))
		}
		var missing []string
		for _, name := range params {
			if name == engine.ContextKeyCurrentTime {
				continue
			}
			if _, ok := t.Condition.Context[name]; !ok {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			return InvalidArgument(fmt.Sprintf("condition %q on tuple %s#%s@%s is missing parameters %s; store every parameter except current_time on the tuple",
				t.Condition.Name, t.Object, t.Relation, t.User, strings.Join(missing, ", ")))
		}
	}
	return nil
}

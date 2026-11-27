package openfga

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	language "github.com/openfga/language/pkg/go/transformer"
	"github.com/openfga/openfga/pkg/tuple"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/internalerr"
	"github.com/authorizerdev/authorizer/internal/metrics"
)

// requestContext builds the OpenFGA request context for an evaluating call:
// the caller's parameters (if any) plus ContextKeyCurrentTime set to the server
// clock. current_time is always overwritten, never taken from the caller — see
// engine.ContextKeyCurrentTime.
func requestContext(params map[string]any) (*structpb.Struct, error) {
	merged := make(map[string]any, len(params)+1)
	for k, v := range params {
		merged[k] = v
	}
	merged[engine.ContextKeyCurrentTime] = time.Now().UTC().Format(time.RFC3339Nano)
	st, err := structpb.NewStruct(merged)
	if err != nil {
		return nil, fmt.Errorf("invalid request context: %w", err)
	}
	return st, nil
}

// toProtoCondition converts an engine.TupleCondition for a write.
func toProtoCondition(c *engine.TupleCondition) (*openfgav1.RelationshipCondition, error) {
	if c == nil {
		return nil, nil
	}
	out := &openfgav1.RelationshipCondition{Name: c.Name}
	if len(c.Context) > 0 {
		st, err := structpb.NewStruct(c.Context)
		if err != nil {
			return nil, fmt.Errorf("invalid condition context: %w", err)
		}
		out.Context = st
	}
	return out, nil
}

// fromProtoCondition converts a stored tuple's condition for a read.
func fromProtoCondition(c *openfgav1.RelationshipCondition) *engine.TupleCondition {
	if c == nil || c.GetName() == "" {
		return nil
	}
	out := &engine.TupleCondition{Name: c.GetName()}
	if c.GetContext() != nil {
		out.Context = c.GetContext().AsMap()
	}
	return out
}

// Check reports whether user is related to object via relation. It is
// fail-closed: any engine error returns (false, err) and callers must treat the
// error as a deny.
func (e *engineImpl) Check(ctx context.Context, user, relation, object string, ctxTuples ...engine.ContextualTuple) (bool, error) {
	storeID, modelID := e.ids()
	if modelID == "" {
		return false, fmt.Errorf("openfga.Check: no authorization model written yet")
	}
	reqCtx, err := requestContext(nil)
	if err != nil {
		return false, err
	}
	res, err := e.srv.Check(ctx, &openfgav1.CheckRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		TupleKey: &openfgav1.CheckRequestTupleKey{
			User:     user,
			Relation: relation,
			Object:   object,
		},
		ContextualTuples: toProtoContextual(ctxTuples),
		Context:          reqCtx,
	})
	if err != nil {
		return false, opErr("openfga.Check", err)
	}
	return res.GetAllowed(), nil
}

// BatchCheck evaluates multiple checks. The returned slice is positionally
// aligned with the input. OpenFGA returns results keyed by a per-item
// correlation ID, so we assign the item index as the correlation ID and map
// back. A whole-batch error fails closed for every request.
func (e *engineImpl) BatchCheck(ctx context.Context, requests []engine.CheckRequest) ([]engine.CheckResult, error) {
	if len(requests) == 0 {
		return nil, nil
	}
	storeID, modelID := e.ids()
	if modelID == "" {
		return nil, fmt.Errorf("openfga.BatchCheck: no authorization model written yet")
	}

	items := make([]*openfgav1.BatchCheckItem, 0, len(requests))
	for i, r := range requests {
		reqCtx, err := requestContext(r.Context)
		if err != nil {
			return nil, err
		}
		items = append(items, &openfgav1.BatchCheckItem{
			TupleKey: &openfgav1.CheckRequestTupleKey{
				User:     r.User,
				Relation: r.Relation,
				Object:   r.Object,
			},
			ContextualTuples: toProtoContextual(r.ContextualTuples),
			Context:          reqCtx,
			CorrelationId:    strconv.Itoa(i),
		})
	}

	res, err := e.srv.BatchCheck(ctx, &openfgav1.BatchCheckRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		Checks:               items,
	})
	if err != nil {
		return nil, opErr("openfga.BatchCheck", err)
	}

	results := make([]engine.CheckResult, len(requests))
	resultMap := res.GetResult()
	for i := range requests {
		single, ok := resultMap[strconv.Itoa(i)]
		if !ok {
			return nil, fmt.Errorf("openfga.BatchCheck: missing result for check %d", i)
		}
		if cerr := single.GetError(); cerr != nil {
			return nil, fmt.Errorf("openfga.BatchCheck: check %d errored: %s", i, cerr.GetMessage())
		}
		results[i] = engine.CheckResult{Allowed: single.GetAllowed()}
	}
	return results, nil
}

// ListObjects returns the IDs of objects of type objType to which user is
// related via relation. This is an expensive enumeration surface — callers must
// paginate/cap/rate-limit at the API boundary.
func (e *engineImpl) ListObjects(ctx context.Context, user, relation, objType string) ([]string, error) {
	storeID, modelID := e.ids()
	if modelID == "" {
		return nil, fmt.Errorf("openfga.ListObjects: no authorization model written yet")
	}
	reqCtx, err := requestContext(nil)
	if err != nil {
		return nil, err
	}
	res, err := e.srv.ListObjects(ctx, &openfgav1.ListObjectsRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		Type:                 objType,
		Relation:             relation,
		User:                 user,
		Context:              reqCtx,
	})
	if err != nil {
		return nil, opErr("openfga.ListObjects", err)
	}
	return res.GetObjects(), nil
}

// ListUsers returns the fully qualified user IDs (e.g. "user:alice") of type
// userType that have relation on object. It is the inverse of ListObjects
// ("who can access this object?") and an expensive enumeration surface that
// reveals the access graph — callers must admin-gate, cap and audit.
func (e *engineImpl) ListUsers(ctx context.Context, object, relation, userType string) ([]string, error) {
	storeID, modelID := e.ids()
	if modelID == "" {
		return nil, fmt.Errorf("openfga.ListUsers: no authorization model written yet")
	}
	objType, objID, found := strings.Cut(object, ":")
	if !found || objType == "" || objID == "" {
		return nil, fmt.Errorf("openfga.ListUsers: object must be in type:id form, got %q", object)
	}
	reqCtx, err := requestContext(nil)
	if err != nil {
		return nil, err
	}
	res, err := e.srv.ListUsers(ctx, &openfgav1.ListUsersRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		Object:               &openfgav1.Object{Type: objType, Id: objID},
		Relation:             relation,
		UserFilters:          []*openfgav1.UserTypeFilter{{Type: userType}},
		Context:              reqCtx,
	})
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpListUsers, metrics.FgaResultError)
		e.log.Debug().Err(err).Msg("openfga: list users failed")
		return nil, opErr("openfga.ListUsers", err)
	}
	metrics.RecordFgaOperation(metrics.FgaOpListUsers, metrics.FgaResultSuccess)
	users := make([]string, 0, len(res.GetUsers()))
	for _, u := range res.GetUsers() {
		users = append(users, string(tuple.UserProtoToString(u)))
	}
	return users, nil
}

// Expand returns the OpenFGA relationship/userset tree for (relation, object)
// rendered as a JSON string (the explainability/"why" primitive). The tree is
// marshaled via protojson so it is a stable, machine-readable representation of
// how the relation resolves.
func (e *engineImpl) Expand(ctx context.Context, relation, object string) (string, error) {
	storeID, modelID := e.ids()
	if modelID == "" {
		return "", fmt.Errorf("openfga.Expand: no authorization model written yet")
	}
	res, err := e.srv.Expand(ctx, &openfgav1.ExpandRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		TupleKey: &openfgav1.ExpandRequestTupleKey{
			Relation: relation,
			Object:   object,
		},
	})
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpExpand, metrics.FgaResultError)
		e.log.Debug().Err(err).Msg("openfga: expand failed")
		return "", opErr("openfga.Expand", err)
	}
	jsonBytes, err := protojson.Marshal(res.GetTree())
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpExpand, metrics.FgaResultError)
		return "", internalerr.Wrap(errorSource, fmt.Errorf("openfga.Expand: marshal tree: %w", err))
	}
	metrics.RecordFgaOperation(metrics.FgaOpExpand, metrics.FgaResultSuccess)
	return string(jsonBytes), nil
}

// WriteTuples persists the given relationship tuples.
func (e *engineImpl) WriteTuples(ctx context.Context, tuples []engine.TupleKey) error {
	if len(tuples) == 0 {
		return nil
	}
	storeID, modelID := e.ids()
	keys := make([]*openfgav1.TupleKey, 0, len(tuples))
	for _, t := range tuples {
		cond, err := toProtoCondition(t.Condition)
		if err != nil {
			return err
		}
		keys = append(keys, &openfgav1.TupleKey{
			User:      t.User,
			Relation:  t.Relation,
			Object:    t.Object,
			Condition: cond,
		})
	}
	_, err := e.srv.Write(ctx, &openfgav1.WriteRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		Writes:               &openfgav1.WriteRequestWrites{TupleKeys: keys},
	})
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpWriteTuples, metrics.FgaResultError)
		e.log.Debug().Err(err).Int("count", len(tuples)).Msg("openfga: write tuples failed")
		return opErr("openfga.WriteTuples", err)
	}
	metrics.RecordFgaOperation(metrics.FgaOpWriteTuples, metrics.FgaResultSuccess)
	return nil
}

// DeleteTuples removes the given relationship tuples.
func (e *engineImpl) DeleteTuples(ctx context.Context, tuples []engine.TupleKey) error {
	if len(tuples) == 0 {
		return nil
	}
	storeID, modelID := e.ids()
	keys := make([]*openfgav1.TupleKeyWithoutCondition, 0, len(tuples))
	for _, t := range tuples {
		keys = append(keys, &openfgav1.TupleKeyWithoutCondition{
			User:     t.User,
			Relation: t.Relation,
			Object:   t.Object,
		})
	}
	_, err := e.srv.Write(ctx, &openfgav1.WriteRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		Deletes:              &openfgav1.WriteRequestDeletes{TupleKeys: keys},
	})
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpDeleteTuples, metrics.FgaResultError)
		e.log.Debug().Err(err).Int("count", len(tuples)).Msg("openfga: delete tuples failed")
		return opErr("openfga.DeleteTuples", err)
	}
	metrics.RecordFgaOperation(metrics.FgaOpDeleteTuples, metrics.FgaResultSuccess)
	return nil
}

// ReadTuples returns a page of persisted tuples matching the filter. An empty
// filter reads all tuples (paginated). The filter's User/Relation/Object map to
// the OpenFGA read tuple-key wildcard semantics.
func (e *engineImpl) ReadTuples(ctx context.Context, filter engine.ReadTuplesFilter) (*engine.ReadTuplesResult, error) {
	storeID, _ := e.ids()

	req := &openfgav1.ReadRequest{
		StoreId:           storeID,
		ContinuationToken: filter.ContinuationToken,
	}
	// Only attach a tuple-key filter when at least one field is set; OpenFGA
	// rejects an empty (all-wildcard) tuple key but allows omitting it to read
	// everything.
	if filter.User != "" || filter.Relation != "" || filter.Object != "" {
		req.TupleKey = &openfgav1.ReadRequestTupleKey{
			User:     filter.User,
			Relation: filter.Relation,
			Object:   filter.Object,
		}
	}
	if filter.PageSize > 0 {
		req.PageSize = wrapperspb.Int32(filter.PageSize)
	}

	res, err := e.srv.Read(ctx, req)
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpReadTuples, metrics.FgaResultError)
		e.log.Debug().Err(err).Msg("openfga: read tuples failed")
		return nil, opErr("openfga.ReadTuples", err)
	}
	metrics.RecordFgaOperation(metrics.FgaOpReadTuples, metrics.FgaResultSuccess)

	out := &engine.ReadTuplesResult{
		ContinuationToken: res.GetContinuationToken(),
		Tuples:            make([]engine.TupleKey, 0, len(res.GetTuples())),
	}
	for _, t := range res.GetTuples() {
		k := t.GetKey()
		out.Tuples = append(out.Tuples, engine.TupleKey{
			User:      k.GetUser(),
			Relation:  k.GetRelation(),
			Object:    k.GetObject(),
			Condition: fromProtoCondition(k.GetCondition()),
		})
	}
	return out, nil
}

// WriteModel installs a new authorization model from its DSL form and returns
// the backend-assigned model ID. The new model becomes the active model for
// subsequent checks. Model writes are powerful and must be admin-gated, audited
// and staged by the caller.
func (e *engineImpl) WriteModel(ctx context.Context, dsl string) (string, error) {
	parsed, err := language.TransformDSLToProto(dsl)
	if err != nil {
		return "", fmt.Errorf("openfga.WriteModel: invalid DSL: %w", err)
	}
	// The engine supplies current_time as an RFC 3339 timestamp on every
	// check. A condition declaring it as another type would fail to evaluate
	// on every check that reaches it, so refuse the model up front.
	for name, cond := range parsed.GetConditions() {
		if p, ok := cond.GetParameters()[engine.ContextKeyCurrentTime]; ok &&
			p.GetTypeName() != openfgav1.ConditionParamTypeRef_TYPE_NAME_TIMESTAMP {
			return "", fmt.Errorf("openfga.WriteModel: condition %q declares %s as %s; it must be a timestamp, which the server supplies",
				name, engine.ContextKeyCurrentTime, strings.TrimPrefix(p.GetTypeName().String(), "TYPE_NAME_"))
		}
	}
	storeID, _ := e.ids()
	wm, err := e.srv.WriteAuthorizationModel(ctx, &openfgav1.WriteAuthorizationModelRequest{
		StoreId:         storeID,
		TypeDefinitions: parsed.GetTypeDefinitions(),
		SchemaVersion:   parsed.GetSchemaVersion(),
		Conditions:      parsed.GetConditions(),
	})
	if err != nil {
		metrics.RecordFgaOperation(metrics.FgaOpWriteModel, metrics.FgaResultError)
		e.log.Debug().Err(err).Msg("openfga: write model failed")
		return "", opErr("openfga.WriteModel", err)
	}
	modelID := wm.GetAuthorizationModelId()

	e.mu.Lock()
	e.modelID = modelID
	e.mu.Unlock()

	metrics.RecordFgaOperation(metrics.FgaOpWriteModel, metrics.FgaResultSuccess)
	e.log.Info().Str("model_id", modelID).Msg("wrote OpenFGA authorization model; persist this ID")
	return modelID, nil
}

// TypeRelations returns the relation names defined on each object type in the
// active model, sorted for deterministic output. Types without relations are
// omitted — they cannot be the object of any permission.
func (e *engineImpl) TypeRelations(ctx context.Context) (map[string][]string, error) {
	storeID, modelID := e.ids()
	if modelID == "" {
		return nil, fmt.Errorf("openfga.TypeRelations: %w", engine.ErrNoModel)
	}
	res, err := e.srv.ReadAuthorizationModel(ctx, &openfgav1.ReadAuthorizationModelRequest{
		StoreId: storeID,
		Id:      modelID,
	})
	if err != nil {
		return nil, opErr("openfga.TypeRelations", err)
	}
	out := make(map[string][]string)
	for _, td := range res.GetAuthorizationModel().GetTypeDefinitions() {
		rels := td.GetRelations()
		if len(rels) == 0 {
			continue
		}
		names := make([]string, 0, len(rels))
		for name := range rels {
			names = append(names, name)
		}
		sort.Strings(names)
		out[td.GetType()] = names
	}
	return out, nil
}

// ReadModel returns the active authorization model: its id and DSL rendering.
func (e *engineImpl) ReadModel(ctx context.Context) (string, string, error) {
	storeID, modelID := e.ids()
	if modelID == "" {
		return "", "", fmt.Errorf("openfga.ReadModel: %w", engine.ErrNoModel)
	}
	res, err := e.srv.ReadAuthorizationModel(ctx, &openfgav1.ReadAuthorizationModelRequest{
		StoreId: storeID,
		Id:      modelID,
	})
	if err != nil {
		return "", "", opErr("openfga.ReadModel", err)
	}
	// Render via the JSON-string transformer rather than the proto-direct
	// TransformJSONProtoToDSL: the latter (language v0.2.1) errors on relations
	// that participate only in a "but not" exclusion (e.g. "not supported by the
	// OpenFGA DSL syntax yet"). The protojson -> JSONString path handles the
	// same model correctly.
	jsonBytes, err := protojson.Marshal(res.GetAuthorizationModel())
	if err != nil {
		return "", "", internalerr.Wrap(errorSource, fmt.Errorf("openfga.ReadModel: marshal model: %w", err))
	}
	dsl, err := language.TransformJSONStringToDSL(string(jsonBytes))
	if err != nil {
		return "", "", internalerr.Wrap(errorSource, fmt.Errorf("openfga.ReadModel: render DSL: %w", err))
	}
	if dsl == nil {
		return "", "", internalerr.Wrap(errorSource, fmt.Errorf("openfga.ReadModel: render DSL returned nil"))
	}
	return modelID, *dsl, nil
}

// TypeNames returns every object type declared in the active model, sorted.
//
// Distinct from TypeRelations, which omits types that declare no relations.
// That omission makes it unusable for detecting a type that is only ever a
// SUBJECT — the canonical `type agent` has no relations of its own, so
// TypeRelations never reports it and any feature keyed on that would silently
// never activate.
func (e *engineImpl) TypeNames(ctx context.Context) ([]string, error) {
	storeID, modelID := e.ids()
	if modelID == "" {
		return nil, fmt.Errorf("openfga.TypeNames: %w", engine.ErrNoModel)
	}
	res, err := e.srv.ReadAuthorizationModel(ctx, &openfgav1.ReadAuthorizationModelRequest{
		StoreId: storeID,
		Id:      modelID,
	})
	if err != nil {
		return nil, opErr("openfga.TypeNames", err)
	}
	defs := res.GetAuthorizationModel().GetTypeDefinitions()
	out := make([]string, 0, len(defs))
	for _, td := range defs {
		if n := td.GetType(); n != "" {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out, nil
}

// ConditionParams returns each condition declared in the active model with its
// sorted parameter names.
func (e *engineImpl) ConditionParams(ctx context.Context) (map[string][]string, error) {
	storeID, modelID := e.ids()
	if modelID == "" {
		return nil, fmt.Errorf("openfga.ConditionParams: %w", engine.ErrNoModel)
	}
	res, err := e.srv.ReadAuthorizationModel(ctx, &openfgav1.ReadAuthorizationModelRequest{
		StoreId: storeID,
		Id:      modelID,
	})
	if err != nil {
		return nil, opErr("openfga.ConditionParams", err)
	}
	out := make(map[string][]string)
	for name, cond := range res.GetAuthorizationModel().GetConditions() {
		params := make([]string, 0, len(cond.GetParameters()))
		for p := range cond.GetParameters() {
			params = append(params, p)
		}
		sort.Strings(params)
		out[name] = params
	}
	return out, nil
}

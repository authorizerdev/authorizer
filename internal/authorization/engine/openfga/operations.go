package openfga

import (
	"context"
	"fmt"
	"strings"

	openfgav1 "github.com/openfga/api/proto/openfga/v1"
	language "github.com/openfga/language/pkg/go/transformer"
	"github.com/openfga/openfga/pkg/tuple"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
)

// Check reports whether user is related to object via relation. It is
// fail-closed: any engine error returns (false, err) and callers must treat the
// error as a deny.
func (e *engineImpl) Check(ctx context.Context, user, relation, object string, ctxTuples ...engine.ContextualTuple) (bool, error) {
	storeID, modelID := e.ids()
	if modelID == "" {
		return false, fmt.Errorf("openfga.Check: no authorization model written yet")
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
	})
	if err != nil {
		return false, fmt.Errorf("openfga.Check: %w", err)
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
		items = append(items, &openfgav1.BatchCheckItem{
			TupleKey: &openfgav1.CheckRequestTupleKey{
				User:     r.User,
				Relation: r.Relation,
				Object:   r.Object,
			},
			ContextualTuples: toProtoContextual(r.ContextualTuples),
			CorrelationId:    strconvItoa(i),
		})
	}

	res, err := e.srv.BatchCheck(ctx, &openfgav1.BatchCheckRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		Checks:               items,
	})
	if err != nil {
		return nil, fmt.Errorf("openfga.BatchCheck: %w", err)
	}

	results := make([]engine.CheckResult, len(requests))
	resultMap := res.GetResult()
	for i := range requests {
		single, ok := resultMap[strconvItoa(i)]
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
	res, err := e.srv.ListObjects(ctx, &openfgav1.ListObjectsRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		Type:                 objType,
		Relation:             relation,
		User:                 user,
	})
	if err != nil {
		return nil, fmt.Errorf("openfga.ListObjects: %w", err)
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
	res, err := e.srv.ListUsers(ctx, &openfgav1.ListUsersRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		Object:               &openfgav1.Object{Type: objType, Id: objID},
		Relation:             relation,
		UserFilters:          []*openfgav1.UserTypeFilter{{Type: userType}},
	})
	if err != nil {
		return nil, fmt.Errorf("openfga.ListUsers: %w", err)
	}
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
		return "", fmt.Errorf("openfga.Expand: %w", err)
	}
	jsonBytes, err := protojson.Marshal(res.GetTree())
	if err != nil {
		return "", fmt.Errorf("openfga.Expand: marshal tree: %w", err)
	}
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
		keys = append(keys, &openfgav1.TupleKey{
			User:     t.User,
			Relation: t.Relation,
			Object:   t.Object,
		})
	}
	_, err := e.srv.Write(ctx, &openfgav1.WriteRequest{
		StoreId:              storeID,
		AuthorizationModelId: modelID,
		Writes:               &openfgav1.WriteRequestWrites{TupleKeys: keys},
	})
	if err != nil {
		return fmt.Errorf("openfga.WriteTuples: %w", err)
	}
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
		return fmt.Errorf("openfga.DeleteTuples: %w", err)
	}
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
		return nil, fmt.Errorf("openfga.ReadTuples: %w", err)
	}

	out := &engine.ReadTuplesResult{
		ContinuationToken: res.GetContinuationToken(),
		Tuples:            make([]engine.TupleKey, 0, len(res.GetTuples())),
	}
	for _, t := range res.GetTuples() {
		k := t.GetKey()
		out.Tuples = append(out.Tuples, engine.TupleKey{
			User:     k.GetUser(),
			Relation: k.GetRelation(),
			Object:   k.GetObject(),
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
	storeID, _ := e.ids()
	wm, err := e.srv.WriteAuthorizationModel(ctx, &openfgav1.WriteAuthorizationModelRequest{
		StoreId:         storeID,
		TypeDefinitions: parsed.GetTypeDefinitions(),
		SchemaVersion:   parsed.GetSchemaVersion(),
		Conditions:      parsed.GetConditions(),
	})
	if err != nil {
		return "", fmt.Errorf("openfga.WriteModel: %w", err)
	}
	modelID := wm.GetAuthorizationModelId()

	e.mu.Lock()
	e.modelID = modelID
	e.mu.Unlock()

	e.log.Info().Str("model_id", modelID).Msg("wrote OpenFGA authorization model; persist this ID")
	return modelID, nil
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
		return "", "", fmt.Errorf("openfga.ReadModel: %w", err)
	}
	// Render via the JSON-string transformer rather than the proto-direct
	// TransformJSONProtoToDSL: the latter (language v0.2.1) errors on relations
	// that participate only in a "but not" exclusion (e.g. "not supported by the
	// OpenFGA DSL syntax yet"). The protojson -> JSONString path handles the
	// same model correctly.
	jsonBytes, err := protojson.Marshal(res.GetAuthorizationModel())
	if err != nil {
		return "", "", fmt.Errorf("openfga.ReadModel: marshal model: %w", err)
	}
	dsl, err := language.TransformJSONStringToDSL(string(jsonBytes))
	if err != nil {
		return "", "", fmt.Errorf("openfga.ReadModel: render DSL: %w", err)
	}
	if dsl == nil {
		return "", "", fmt.Errorf("openfga.ReadModel: render DSL returned nil")
	}
	return modelID, *dsl, nil
}

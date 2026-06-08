// Package engine defines the AuthorizationEngine SPI — the abstraction over a
// relationship-based access control (ReBAC) backend used by Authorizer's
// fine-grained authorization (FGA) subsystem.
//
// The interface is deliberately backend-agnostic. The Phase 1 implementation
// (internal/authorization/engine/openfga) embeds OpenFGA in-process, but the
// same contract is intended to also front an external OpenFGA service. The
// engine speaks the OpenFGA tuple vocabulary: a tuple relates a user (subject)
// to an object via a relation, e.g. (user:alice, viewer, document:1).
//
// This package is additive (Phase 1 of the OpenFGA migration). It does not
// replace the existing authorization.Provider (resource/scope/policy engine);
// both coexist behind the --authorization-engine flag.
package engine

import "context"

// TupleKey identifies a single relationship: the subject (User) is related to
// the Object via the Relation. Identifiers are expected to be fully qualified
// in OpenFGA form, e.g. User="user:alice", Relation="viewer",
// Object="document:1". The User field may also be a userset reference such as
// "role:admin#assignee".
type TupleKey struct {
	// User is the subject of the relationship (e.g. "user:alice" or a userset
	// like "role:admin#assignee").
	User string
	// Relation is the relation name connecting the user to the object (e.g.
	// "viewer").
	Relation string
	// Object is the fully qualified object (e.g. "document:1").
	Object string
}

// ContextualTuple is a tuple supplied only for the duration of a single Check
// or BatchCheck call. It is not persisted; it lets callers evaluate
// hypothetical or request-scoped relationships (the OpenFGA contextual-tuples
// feature) without writing to the store.
type ContextualTuple struct {
	// User is the subject of the relationship.
	User string
	// Relation is the relation name.
	Relation string
	// Object is the fully qualified object.
	Object string
}

// CheckRequest is a single relationship-check question: "is User related to
// Object via Relation?". ContextualTuples, if any, are evaluated alongside the
// persisted tuples for this check only.
type CheckRequest struct {
	// User is the subject being checked (e.g. "user:alice").
	User string
	// Relation is the relation to evaluate (e.g. "can_view").
	Relation string
	// Object is the fully qualified object (e.g. "document:1").
	Object string
	// ContextualTuples are request-scoped tuples not persisted to the store.
	ContextualTuples []ContextualTuple
}

// CheckResult is the answer to a single CheckRequest.
type CheckResult struct {
	// Allowed reports whether the relationship holds.
	Allowed bool
}

// ReadTuplesFilter narrows a ReadTuples query. Any field left empty acts as a
// wildcard for that position. An entirely empty filter reads all tuples (use
// with pagination; this is an enumeration surface).
type ReadTuplesFilter struct {
	// User filters by subject (optional).
	User string
	// Relation filters by relation (optional).
	Relation string
	// Object filters by object (optional); may be a type prefix like
	// "document:" depending on backend support.
	Object string
	// PageSize caps the number of tuples returned in one page. Zero lets the
	// backend choose a default.
	PageSize int32
	// ContinuationToken resumes a previous ReadTuples call; empty starts from
	// the beginning.
	ContinuationToken string
}

// ReadTuplesResult is one page of tuples plus a continuation token for the
// next page (empty when exhausted).
type ReadTuplesResult struct {
	// Tuples is the page of matching relationships.
	Tuples []TupleKey
	// ContinuationToken, when non-empty, can be passed back via
	// ReadTuplesFilter.ContinuationToken to fetch the next page.
	ContinuationToken string
}

// AuthorizationEngine is the SPI for a ReBAC authorization backend.
//
// All decision methods (Check, BatchCheck, ListObjects) are fail-closed at the
// call site: callers must treat a non-nil error as a deny and never as an
// allow. Identifiers follow OpenFGA conventions ("type:id", relation names,
// usersets "type:id#relation").
//
// Implementations are expected to be safe for concurrent use.
type AuthorizationEngine interface {
	// Check reports whether user is related to object via relation. Optional
	// contextual tuples are evaluated for this call only and are not persisted.
	// Returns (false, err) on engine error; callers must fail closed.
	Check(ctx context.Context, user, relation, object string, ctxTuples ...ContextualTuple) (bool, error)

	// BatchCheck evaluates multiple CheckRequests. The returned slice is
	// positionally aligned with the input: result[i] answers requests[i]. An
	// error indicates a whole-batch failure; callers must fail closed for every
	// request in the batch.
	BatchCheck(ctx context.Context, requests []CheckRequest) ([]CheckResult, error)

	// ListObjects returns the IDs of objects of type objType to which user is
	// related via relation. This is the RAG/pre-filter primitive and is an
	// expensive enumeration surface — callers must paginate, cap, and
	// rate-limit. Returned IDs are fully qualified ("document:1").
	ListObjects(ctx context.Context, user, relation, objType string) ([]string, error)

	// ListUsers returns the fully qualified user IDs (e.g. "user:alice") of type
	// userType that have relation on object. It is the inverse of ListObjects:
	// "who can access this object?". This is a powerful enumeration surface that
	// reveals the access graph — callers must admin-gate, cap and audit. Returned
	// users are fully qualified ("user:alice").
	ListUsers(ctx context.Context, object, relation, userType string) ([]string, error)

	// Expand returns the OpenFGA relationship/userset tree for (relation, object)
	// rendered as a JSON string. This is the explainability/"why" primitive: it
	// shows how a relation resolves (direct assignments, usersets, computed
	// relations). It reveals the access graph and must be admin-gated.
	Expand(ctx context.Context, relation, object string) (string, error)

	// WriteTuples persists the given relationship tuples. It is additive;
	// duplicate writes may error depending on the backend.
	WriteTuples(ctx context.Context, tuples []TupleKey) error

	// DeleteTuples removes the given relationship tuples. Deleting a
	// non-existent tuple may error depending on the backend.
	DeleteTuples(ctx context.Context, tuples []TupleKey) error

	// ReadTuples returns a page of persisted tuples matching the filter, plus a
	// continuation token. It is an enumeration surface — always paginate.
	ReadTuples(ctx context.Context, filter ReadTuplesFilter) (*ReadTuplesResult, error)

	// WriteModel installs a new authorization model from its DSL form and
	// returns the backend-assigned model ID. Writing a model is powerful (a
	// single edit can re-grant broadly) and must be admin-gated, audited, and
	// staged by callers.
	WriteModel(ctx context.Context, dsl string) (string, error)

	// ReadModel returns the currently active authorization model: its
	// backend-assigned id and its DSL rendering.
	ReadModel(ctx context.Context) (id string, dsl string, err error)
}

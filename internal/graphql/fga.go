package graphql

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/graph/model"
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

// maxFgaListResults caps the number of objects returned by list_permissions
// and the page size of admin tuple reads. Listing is an expensive enumeration
// surface, so the result set is bounded.
const maxFgaListResults = 1000

// resolveFgaSubject is the single, centralized trust gate for the public
// permission APIs (check_permissions, list_permissions). It decides which
// OpenFGA subject ("type:id") a decision is evaluated for, given the optional
// client-supplied explicitUser.
//
// Rules (fail-closed):
//   - explicitUser empty → the caller's own token subject ("user:<sub>" from
//     the JWT / session cookie). This is the default and the common case.
//   - explicitUser set → normalized (a bare id becomes "user:<id>") and
//     validated, then honored only when the caller is a super-admin OR the
//     subject equals the caller's own token subject. Anything else is
//     REJECTED — never silently self-pinned, never silently ignored —
//     because honoring it would let an end user probe another subject's
//     access (IDOR / info disclosure).
//

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
	return fmt.Errorf("invalid tuple %q: %s — the relation and object type must be defined in the active authorization model (Step 1)", m[1], m[2])
}

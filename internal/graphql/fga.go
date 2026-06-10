package graphql

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/graph/model"
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

// maxFgaListResults caps the number of objects returned by list_permissions
// and the page size of admin tuple reads. Listing is an expensive enumeration
// surface, so the result set is bounded.
const maxFgaListResults = 1000

// maxPermissionChecks caps the number of checks accepted by a single
// check_permissions call.
const maxPermissionChecks = 100

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
// TODO(phase-2 M2M): machine-to-machine / client-credentials callers should
// also be allowed to pass an explicit user once that caller type exists;
// extend the trust check here (the rule must stay centralized in this one
// helper).
func (g *graphqlProvider) resolveFgaSubject(ctx context.Context, explicitUser string) (string, error) {
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return "", err
	}
	explicitUser = strings.TrimSpace(explicitUser)

	// The caller's own subject, when they carry a user token/session. Resolved
	// lazily-ish here because both branches may need it.
	ownSubject := ""
	if tokenData, terr := g.TokenProvider.GetUserIDFromSessionOrAccessToken(gc); terr == nil && strings.TrimSpace(tokenData.UserID) != "" {
		ownSubject = "user:" + tokenData.UserID
	}

	if explicitUser == "" {
		// Default: pin to the caller's own token subject.
		if ownSubject == "" {
			return "", fmt.Errorf("unauthorized")
		}
		return ownSubject, nil
	}

	subject := normalizeFgaSubject(explicitUser)
	if err := validateFgaSubject(subject); err != nil {
		return "", err
	}
	// Self-specification is always allowed: it is exactly what the token
	// already proves. This keeps client code symmetric (it may always send
	// `user`) while the server stays strict.
	if subject == ownSubject {
		return subject, nil
	}
	// Only a super-admin may evaluate a different subject. The trust level is
	// derived from the admin cookie/secret — never from client input.
	if g.TokenProvider.IsSuperAdmin(gc) {
		return subject, nil
	}
	return "", fmt.Errorf("not authorized to query authorization for another subject")
}

// normalizeFgaSubject turns a bare id into the canonical "user:<id>" form;
// values that already carry a type ("type:id") pass through unchanged.
func normalizeFgaSubject(user string) string {
	if !strings.Contains(user, ":") {
		return "user:" + user
	}
	return user
}

// validateFgaSubject ensures an explicitly supplied subject is in OpenFGA
// "type:id" form (both halves non-empty). It rejects usersets
// ("type:id#relation") and malformed values.
func validateFgaSubject(user string) error {
	objType, objID, found := strings.Cut(user, ":")
	if !found || strings.TrimSpace(objType) == "" || strings.TrimSpace(objID) == "" {
		return fmt.Errorf("user must be in type:id form, got %q", user)
	}
	if strings.Contains(objID, "#") {
		return fmt.Errorf("user must be a concrete subject in type:id form, not a userset, got %q", user)
	}
	return nil
}

// toContextualTuples converts client-supplied contextual tuples. These are
// request-scoped only (never persisted) and are safe to accept from the client.
func toContextualTuples(in []*model.FgaTupleInput) ([]engine.ContextualTuple, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]engine.ContextualTuple, 0, len(in))
	for _, t := range in {
		if t == nil || strings.TrimSpace(t.User) == "" || strings.TrimSpace(t.Relation) == "" || strings.TrimSpace(t.Object) == "" {
			return nil, fmt.Errorf("each contextual tuple requires user, relation and object")
		}
		out = append(out, engine.ContextualTuple{User: t.User, Relation: t.Relation, Object: t.Object})
	}
	return out, nil
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

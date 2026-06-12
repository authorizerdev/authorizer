package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// ErrFgaNotEnabled is returned by every fine-grained-authorization (FGA)
// operation when no authorization engine is configured (no --fga-store).
// Fail-closed.
var ErrFgaNotEnabled = errors.New("fine-grained authorization is not enabled")

// maxFgaListResults caps the number of permissions returned by
// ListPermissions. Listing is an expensive enumeration surface, so the
// result set is bounded and `truncated` reports when the cap was hit.
const maxFgaListResults = 1000

// maxPermissionChecks caps the number of checks accepted by a single
// CheckPermissions call.
const maxPermissionChecks = 100

// maxContextualTuplesPerCheck caps client-supplied contextual tuples on a
// single check. OpenFGA enforces its own per-request limit (default 100), but
// the boundary must not depend on the embedded engine's configuration —
// contextual tuples are accepted from any authenticated caller.
const maxContextualTuplesPerCheck = 100

// resolveFgaSubject is the single, centralized trust gate for the public
// permission APIs (CheckPermissions, ListPermissions). It decides which
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
func (p *provider) resolveFgaSubject(meta RequestMetadata, explicitUser string) (string, error) {
	gc := &gin.Context{Request: meta.Request}
	explicitUser = strings.TrimSpace(explicitUser)

	// The caller's own subject, when they carry a user token/session. Resolved
	// lazily-ish here because both branches may need it.
	ownSubject := ""
	if tokenData, terr := p.TokenProvider.GetUserIDFromSessionOrAccessToken(gc); terr == nil && strings.TrimSpace(tokenData.UserID) != "" {
		ownSubject = "user:" + tokenData.UserID
	}

	if explicitUser == "" {
		// Default: pin to the caller's own token subject.
		if ownSubject == "" {
			return "", Unauthenticated("unauthorized")
		}
		return ownSubject, nil
	}

	subject := normalizeFgaSubject(explicitUser)
	if err := validateFgaSubject(subject); err != nil {
		return "", err
	}
	// Self-specification is always allowed: it is exactly what the token
	// already proves. This keeps client code symmetric (it may always send
	// `user`) while the server stays strict. The comparison is exact-string
	// after outer TrimSpace + normalization — no inner-whitespace or case
	// tolerance; a near-miss falls through and is rejected (fail-closed).
	if subject == ownSubject {
		return subject, nil
	}
	// Only a super-admin may evaluate a different subject. The trust level is
	// derived from the admin cookie/secret — never from client input.
	if p.TokenProvider.IsSuperAdmin(gc) {
		return subject, nil
	}
	return "", PermissionDenied("not authorized to query authorization for another subject")
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
		return InvalidArgument(fmt.Sprintf("user must be in type:id form, got %q", user))
	}
	if strings.Contains(objID, "#") {
		return InvalidArgument(fmt.Sprintf("user must be a concrete subject in type:id form, not a userset, got %q", user))
	}
	return nil
}

// toContextualTuples converts client-supplied contextual tuples. These are
// request-scoped only (never persisted) and are safe to accept from any
// authenticated caller, but the count is capped so a single check cannot
// carry an unbounded payload.
func toContextualTuples(in []*model.FgaTupleInput) ([]engine.ContextualTuple, error) {
	if len(in) == 0 {
		return nil, nil
	}
	if len(in) > maxContextualTuplesPerCheck {
		return nil, InvalidArgument(fmt.Sprintf("too many contextual tuples: max %d per check", maxContextualTuplesPerCheck))
	}
	out := make([]engine.ContextualTuple, 0, len(in))
	for _, t := range in {
		if t == nil || strings.TrimSpace(t.User) == "" || strings.TrimSpace(t.Relation) == "" || strings.TrimSpace(t.Object) == "" {
			return nil, InvalidArgument("each contextual tuple requires user, relation and object")
		}
		out = append(out, engine.ContextualTuple{
			User:     t.User,
			Relation: t.Relation,
			Object:   t.Object,
		})
	}
	return out, nil
}

// enforceRequiredRelations gates a request on fine-grained authorization. For
// each required (relation, object) it asks the engine whether the caller
// (subject "user:<userID>") holds that relation. Semantics:
//
//   - AND: every relation must be allowed.
//   - Fail-closed: an engine error OR any deny => "unauthorized".
//   - Empty list => authorized (preserves the prior common-case behavior where
//     no fine-grained gating was requested).
//   - Non-empty list with a nil engine => error (FGA not enabled but required).
//
// The subject is always derived server-side from the resolved userID, never
// from client input.
func (p *provider) enforceRequiredRelations(ctx context.Context, log zerolog.Logger, userID string, required []*model.FgaRelationInput) error {
	if len(required) == 0 {
		return nil
	}
	if p.AuthzEngine == nil {
		return ErrFgaNotEnabled
	}
	if strings.TrimSpace(userID) == "" {
		return Unauthenticated("unauthorized")
	}
	subject := "user:" + userID
	for _, r := range required {
		if r == nil || strings.TrimSpace(r.Relation) == "" || strings.TrimSpace(r.Object) == "" {
			return InvalidArgument("each required relation needs relation and object")
		}
		allowed, err := p.AuthzEngine.Check(ctx, subject, r.Relation, r.Object)
		if err != nil {
			// Fail closed.
			log.Debug().Err(err).Str("relation", r.Relation).Str("object", r.Object).Msg("required relation check errored")
			return PermissionDenied("unauthorized")
		}
		if !allowed {
			log.Debug().Str("relation", r.Relation).Str("object", r.Object).Msg("required relation denied")
			return PermissionDenied("unauthorized")
		}
	}
	return nil
}

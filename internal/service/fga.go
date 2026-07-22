package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/authctx"
	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
)

// ErrFgaNotEnabled is returned by every fine-grained-authorization (FGA)
// operation when no authorization engine is configured (no --fga-store).
// Fail-closed. Typed FailedPrecondition so gRPC/REST callers get
// codes.FailedPrecondition / HTTP 400 instead of an internal error.
var ErrFgaNotEnabled = FailedPrecondition("fine-grained authorization is not enabled")

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
//   - explicitUser empty → the caller's own subject (see callerOwnSubject):
//     "user:<sub>" for a human/session caller, or "service_account:<client_id>"
//     for an autonomous client_credentials (machine) caller. This is the
//     default and the common case.
//   - explicitUser set → normalized (a bare id becomes "user:<id>") and
//     validated, then honored only when the caller is a super-admin OR the
//     subject equals the caller's own subject. Anything else is REJECTED —
//     never silently self-pinned, never silently ignored — because honoring it
//     would let a caller probe another subject's access (IDOR / info
//     disclosure). A machine caller may therefore only self-pin (its
//     "service_account:<client_id>") or be denied; it is never a super-admin.
func (p *provider) resolveFgaSubject(ctx context.Context, meta RequestMetadata, explicitUser string) (string, error) {
	explicitUser = strings.TrimSpace(explicitUser)

	// The caller's own subject, when they carry a user/session/machine token.
	// Fail-closed: a machine token whose client cannot be resolved errors here.
	ownSubject, err := p.callerOwnSubject(ctx, meta)
	if err != nil {
		return "", err
	}

	if explicitUser == "" {
		// Default: pin to the caller's own subject.
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
	if principal, ok := authctx.FromContext(ctx); ok && principal.IsSuperAdmin {
		return subject, nil
	}
	gc := &gin.Context{Request: meta.Request}
	if p.TokenProvider.IsSuperAdmin(gc) {
		return subject, nil
	}
	return "", PermissionDenied("not authorized to query authorization for another subject")
}

// callerOwnSubject returns the caller's canonical OpenFGA subject derived from
// their authenticated token/session, or "" when the request carries no user or
// machine credential (e.g. a super admin authenticated only by the admin
// cookie/secret). It is the single place that classifies a caller as a machine
// (client_credentials) subject vs a human user subject.
//
// MACHINE vs USER vs DELEGATED — the classification keys ONLY on the token's
// login_method claim:
//   - login_method == constants.AuthRecipeMethodServiceAccount is stamped
//     EXCLUSIVELY on client_credentials machine tokens
//     (token.createMachineAccessToken). Those tokens have no resource-owner user
//     (sub is the service account's surrogate id) and never carry an RFC 8693
//     `act` delegation claim. Such a caller resolves to
//     "service_account:<client_id>".
//   - every other login_method (human recipes, sso) resolves to "user:<sub>".
//
// This makes the delegation guard structural, not a runtime check: an RFC 8693
// delegated token (token.CreateDelegatedAccessToken) is stateless, carries a
// user `sub` plus an `act` chain, and carries NO login_method claim — so it can
// never be classified as a machine subject and always resolves to "user:<sub>".
// The security-critical rule (delegated and user tokens stay user subjects; only
// autonomous machine tokens become service_account subjects) holds by
// construction.
func (p *provider) callerOwnSubject(ctx context.Context, meta RequestMetadata) (string, error) {
	callerID, loginMethod := "", ""
	if principal, ok := authctx.FromContext(ctx); ok && strings.TrimSpace(principal.UserID) != "" {
		callerID = principal.UserID
		loginMethod = principal.LoginMethod
	} else {
		gc := &gin.Context{Request: meta.Request}
		if tokenData, terr := p.TokenProvider.GetUserIDFromSessionOrAccessToken(gc); terr == nil && strings.TrimSpace(tokenData.UserID) != "" {
			callerID = tokenData.UserID
			loginMethod = tokenData.LoginMethod
		}
	}
	if callerID == "" {
		return "", nil
	}
	if loginMethod == constants.AuthRecipeMethodServiceAccount {
		return p.machineFgaSubject(ctx, callerID)
	}
	return "user:" + callerID, nil
}

// machineFgaSubject maps an authenticated client_credentials caller — whose
// token `sub` is the service account's SURROGATE id (schemas.Client.ID) — to its
// OpenFGA subject "service_account:<client_id>". It resolves the PUBLIC client_id
// so that admin-written tuples and the client registry share one key (locked
// decision: reuse client_id, never the internal surrogate id).
//
// Fail-closed: any lookup failure — or a client whose kind is not
// service_account — denies rather than falling back to any other subject. A
// machine caller is therefore NEVER silently promoted to a user subject.
//
// If the deployment's authorization model does not declare the service_account
// type, the downstream engine Check/ListObjects on this subject fails closed
// (error or no-match deny) — the caller can never inherit a user's access.
func (p *provider) machineFgaSubject(ctx context.Context, serviceAccountID string) (string, error) {
	client, err := p.StorageProvider.GetClientByID(ctx, serviceAccountID)
	if err != nil || client == nil {
		return "", PermissionDenied("unauthorized")
	}
	// Defense in depth: only service_account clients are FGA subjects; an
	// interactive client must never become one. Machine tokens are only ever
	// issued to service_account clients, so this is a belt-and-suspenders guard.
	if client.Kind != constants.ClientKindServiceAccount {
		return "", PermissionDenied("unauthorized")
	}
	// FGA is an authorization decision surface: deny deactivated service
	// accounts here even though their tokens stay valid until exp elsewhere
	// (issuance already blocks new tokens; this gives revocation teeth where
	// it matters most).
	if !client.IsActive {
		return "", PermissionDenied("unauthorized")
	}
	clientID := strings.TrimSpace(client.ClientID)
	if clientID == "" {
		// Legacy rows may carry an empty client_id; storage defaults it to ID.
		clientID = client.ID
	}
	// Defense in depth: client_id is a server-generated UUID today, but the
	// subject string must never smuggle tuple syntax if that ever changes.
	if strings.ContainsAny(clientID, ":#@ \t\n") {
		return "", PermissionDenied("unauthorized")
	}
	return "service_account:" + clientID, nil
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
		start := time.Now()
		allowed, err := p.AuthzEngine.Check(ctx, subject, r.Relation, r.Object)
		metrics.ObserveFgaCheckDuration(metrics.FgaOpRequiredRelations, time.Since(start).Seconds())
		if err != nil {
			// Fail closed.
			metrics.RecordFgaCheck(metrics.FgaOpRequiredRelations, metrics.FgaResultError)
			log.Debug().Err(err).Str("relation", r.Relation).Str("object", r.Object).Msg("required relation check errored")
			return PermissionDenied("unauthorized")
		}
		metrics.RecordFgaCheckResult(metrics.FgaOpRequiredRelations, allowed)
		if !allowed {
			log.Debug().Str("relation", r.Relation).Str("object", r.Object).Msg("required relation denied")
			return PermissionDenied("unauthorized")
		}
	}
	return nil
}

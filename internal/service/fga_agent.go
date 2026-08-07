package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/metrics"
)

// FgaAgentSubjectType is the OpenFGA object type an agent is represented as
// when the operator's model declares it.
//
// Chosen to match the shape published by OpenFGA and Auth0 FGA for AI agents,
// where an agent is a first-class principal appearing wherever `user` does:
//
//	type agent
//
//	type document
//	  relations
//	    define viewer: [user, agent]
const FgaAgentSubjectType = "agent"

// agentModelTTL bounds how long the "does the active model declare an agent
// type" answer is cached when it could not be tied to a model id. The normal
// path keys the cache on the model id itself and needs no expiry.
const agentModelTTL = 30 * time.Second

// agentSubjectsState caches whether the active authorization model declares the
// agent type.
//
// Detection is per-model, not per-request: TypeNames reads the model from the
// datastore, which is far too expensive to do on every permission check. The
// cache is keyed on the model id, so a model write (which mints a new id, and
// which replicas converge on via the engine's refresh) invalidates it for free.
type agentSubjectsState struct {
	mu        sync.RWMutex
	modelID   string
	enabled   bool
	checkedAt time.Time
}

// agentSubjectsEnabled reports whether delegated tokens should be authorized as
// `agent:<client_id>` intersected with `user:<sub>`, rather than as the user
// alone.
//
// THIS IS THE OPT-IN. There is no flag: declaring `type agent` in the
// authorization model IS the operator's opt-in, because the feature is
// meaningless without a model that can express agent grants. A deployment whose
// model has no agent type keeps today's behaviour byte-for-byte.
//
// That choice is forced by how OpenFGA fails. Checking `agent:x` against a
// model with no agent type does not return false — it ERRORS with
// "invalid relation: type 'agent' not found", and CheckPermissions fails closed
// on engine errors. Enabling this against an unprepared model would therefore
// deny EVERY permission check, a total authorization outage rather than a
// graceful degradation. Auto-detection makes that state unreachable.
//
// Detection is only consulted for a DELEGATED caller, so the blast radius of
// an error here is limited to agent traffic. That is what makes failing CLOSED
// affordable: a model read that fails must DENY the delegated request rather
// than quietly fall back to authorizing the user alone. The fallback is the
// dangerous direction — it drops the agent half of the intersection, which is
// the entire protection, and it is reachable by anyone who can make the model
// read fail. An unreachable datastore fails the subsequent Check anyway, so
// denying here costs nothing that was going to succeed.
func (p *provider) agentSubjectsEnabled(ctx context.Context) (bool, error) {
	if p.AuthzEngine == nil {
		return false, ErrFgaNotEnabled
	}

	modelID, _, err := p.AuthzEngine.ReadModel(ctx)
	if err != nil {
		return false, err
	}

	p.agentSubjects.mu.RLock()
	cachedID, cachedEnabled, checkedAt := p.agentSubjects.modelID, p.agentSubjects.enabled, p.agentSubjects.checkedAt
	p.agentSubjects.mu.RUnlock()

	if modelID != "" && cachedID == modelID {
		return cachedEnabled, nil
	}
	if modelID == "" && !checkedAt.IsZero() && time.Since(checkedAt) < agentModelTTL {
		return cachedEnabled, nil
	}

	names, err := p.AuthzEngine.TypeNames(ctx)
	if err != nil {
		return false, err
	}
	enabled := false
	for _, n := range names {
		if n == FgaAgentSubjectType {
			enabled = true
			break
		}
	}

	p.agentSubjects.mu.Lock()
	p.agentSubjects.modelID = modelID
	p.agentSubjects.enabled = enabled
	p.agentSubjects.checkedAt = time.Now()
	p.agentSubjects.mu.Unlock()

	return enabled, nil
}

// delegationSubjects returns the subjects a permission check must satisfy for
// the calling principal.
//
// For an ordinary caller this is a single subject and behaviour is unchanged.
// For an RFC 8693 delegated token — an agent acting for a user — it is BOTH
// `agent:<client_id>` and `user:<sub>`, and every returned subject must be
// allowed for the action to be permitted.
//
// That intersection is the fix for the Confused Deputy problem: an agent
// holding a broad grant must not be able to act on a resource its delegating
// user cannot reach, and equally must not inherit the user's full reach just
// because it holds their token. Effective authority is
// perms(agent) ∩ perms(user), evaluated per action at request time.
//
// Only the IMMEDIATE actor participates. Prior actors nested deeper in the
// `act` chain are informational and must not influence the decision — they were
// asserted upstream, not verified here.
//
// An error DENIES the request. It is never "authorize the user alone": that
// silently discards the agent half and is precisely the Confused Deputy this
// exists to prevent.
//
// `subject` is whatever the trust gate resolved — for a delegated caller
// resolveFgaSubject guarantees that is the delegating user's own subject, so
// the agent half can never be shed by naming a subject explicitly.
func (p *provider) delegationSubjects(ctx context.Context, caller fgaCaller, subject, operation string) ([]string, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil, Unauthenticated("unauthorized")
	}
	if !caller.isDelegated() {
		// The overwhelmingly common path: one subject, zero model reads,
		// behaviour bit-for-bit identical to before delegation existed.
		return []string{subject}, nil
	}

	// Defense in depth. actorID is a server-generated client_id today, but this
	// value is about to be concatenated into an OpenFGA subject string, and a
	// separator smuggled in there would address a different subject entirely
	// (e.g. "agent:x#member" is a userset, not a concrete agent). Mirrors the
	// identical guard in machineFgaSubject.
	actorID := caller.actorID
	if strings.ContainsAny(actorID, ":#@ \t\n") {
		return nil, PermissionDenied("unauthorized")
	}

	enabled, err := p.agentSubjectsEnabled(ctx)
	if err != nil {
		return nil, PermissionDenied("authorization check failed")
	}
	if !enabled {
		// The model cannot express agent grants, so the agent half of
		// perms(agent) ∩ perms(user) cannot be evaluated at all. (Checking
		// `agent:x` against a model with no agent type ERRORS rather than
		// returning false, so there is no "just evaluate it anyway" option.)
		//
		// Fail closed. Authorizing as the user alone silently hands the agent
		// the user's full authority — exactly the Confused Deputy this function
		// exists to prevent — and it does so invisibly, at the moment the
		// control is least able to defend itself. A security check that cannot
		// be evaluated is not a check that passes.
		//
		// The remedy is to add `type agent` to the authorization model. An
		// operator who needs the old behaviour while migrating can set
		// --fga-allow-unconstrained-agents, which is metered so the exposure is
		// visible in dashboards rather than discovered during an incident.
		metrics.RecordFgaDelegatedCheck(operation, metrics.FgaDelegatedNotEnforced)
		// Nil Config means an unconfigured provider, which must not read as
		// "the operator opted out" — and must not nil-panic in a request path
		// either, since an unrecovered panic here takes the process down.
		allowUnconstrained := p.Config != nil && p.Config.FgaAllowUnconstrainedAgents
		if !allowUnconstrained {
			p.logWarn().
				Str("operation", operation).
				Str("agent", actorID).
				Msg("denied delegated FGA check: the authorization model has no `type agent`, so the agent half of the permission intersection cannot be evaluated. Add `type agent` to the model, or set --fga-allow-unconstrained-agents to authorize as the delegating user alone")
			return nil, PermissionDenied("unauthorized")
		}
		p.logWarn().
			Str("operation", operation).
			Str("agent", actorID).
			Msg("delegated FGA check is NOT enforcing the agent constraint: the authorization model has no `type agent` and --fga-allow-unconstrained-agents is set, so this agent carries the delegating user's full authority")
		return []string{subject}, nil
	}

	// Agent first: it is the cheaper, more selective denial, and a denied agent
	// short-circuits before the user check runs.
	return []string{FgaAgentSubjectType + ":" + actorID, subject}, nil
}

// logWarn returns a warn-level event, tolerating a provider built without a
// logger (unit tests, and any future wiring that omits it). A nil *zerolog.Logger
// here would panic inside an authorization decision, turning a missing
// dependency into an outage.
func (p *provider) logWarn() *zerolog.Event {
	if p.Log == nil {
		nop := zerolog.Nop()
		return nop.Warn()
	}
	return p.Log.Warn()
}

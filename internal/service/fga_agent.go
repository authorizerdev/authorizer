package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/authorizerdev/authorizer/internal/authctx"
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
// Fails safe in both directions: any error resolving the model leaves agent
// subjects OFF, which is the current, working behaviour.
func (p *provider) agentSubjectsEnabled(ctx context.Context) bool {
	if p.AuthzEngine == nil {
		return false
	}

	modelID, _, err := p.AuthzEngine.ReadModel(ctx)
	if err != nil {
		// No model, or the store is unreachable. Either way: behave as today.
		return false
	}

	p.agentSubjects.mu.RLock()
	cachedID, cachedEnabled, checkedAt := p.agentSubjects.modelID, p.agentSubjects.enabled, p.agentSubjects.checkedAt
	p.agentSubjects.mu.RUnlock()

	if modelID != "" && cachedID == modelID {
		return cachedEnabled
	}
	if modelID == "" && !checkedAt.IsZero() && time.Since(checkedAt) < agentModelTTL {
		return cachedEnabled
	}

	names, err := p.AuthzEngine.TypeNames(ctx)
	if err != nil {
		return false
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

	return enabled
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
// Returns nil when there is no authenticated caller, which callers treat as
// unauthenticated rather than as "allow".
func (p *provider) delegationSubjects(ctx context.Context, ownSubject string) []string {
	ownSubject = strings.TrimSpace(ownSubject)
	if ownSubject == "" {
		return nil
	}

	principal, ok := authctx.FromContext(ctx)
	if !ok || !principal.IsDelegated() {
		return []string{ownSubject}
	}
	if !p.agentSubjectsEnabled(ctx) {
		// Model cannot express agent grants; preserve existing behaviour.
		return []string{ownSubject}
	}

	agentSubject := FgaAgentSubjectType + ":" + strings.TrimSpace(principal.ActorID)
	if agentSubject == FgaAgentSubjectType+":" {
		return []string{ownSubject}
	}
	// Agent first: it is the cheaper, more selective denial, and a denied agent
	// short-circuits before the user check runs.
	return []string{agentSubject, ownSubject}
}

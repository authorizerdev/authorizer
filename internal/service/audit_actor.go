package service

import (
	"fmt"
	"strings"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
)

// applyDelegationActor rewrites an audit event so that, when the caller is an
// AI agent acting on behalf of a user, the AGENT is recorded as the actor and
// the user it acted for is preserved alongside.
//
// Without this an agent's action is indistinguishable from the user performing
// it themselves: same actor id, same actor type, no trace that anything
// automated was involved. An agent doing something damaging would read as the
// human doing it, silently — a compliance and incident-response problem, and
// one that cannot be fixed after the fact because the information was never
// written.
//
// RFC 8693 §1.1 draws exactly this line: delegation is "A representing B" where
// A keeps its own identity, as opposed to impersonation where A "is
// indistinguishable from B". The audit trail has to be able to tell them apart.
//
// Shape of the rewrite:
//
//	ActorID      -> the agent's client_id (from the token's `act.sub`)
//	ActorType    -> constants.AuditActorTypeAgent
//	ActorEmail   -> cleared; an agent has no mailbox, and leaving the user's
//	                address there is precisely the confusion being removed
//	Metadata     -> gains "delegated_user_id" (and the user's email when the
//	                event carried one), so "who did this, and for whom" both
//	                survive
//
// An empty actorID — an ordinary, non-delegated caller — returns the event
// unchanged, so every existing call site keeps its current behaviour exactly.
//
// The actor is passed in rather than read from the context on purpose. It used
// to come from authctx.Principal, which ONLY the gRPC interceptor populates, so
// on GraphQL every agent action was attributed to the human and the whole
// mechanism was dead on the primary surface. Callers already hold the resolved
// caller identity (callerTokenData), which knows the actor on every transport;
// taking it from there makes the attribution transport-independent by
// construction rather than by a second lookup that can drift again.
func applyDelegationActor(actorID string, event audit.Event) audit.Event {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return event
	}

	delegatedUserID := event.ActorID
	delegatedEmail := event.ActorEmail

	event.ActorID = actorID
	event.ActorType = constants.AuditActorTypeAgent
	event.ActorEmail = ""
	event.Metadata = mergeAuditMetadata(event.Metadata, delegatedUserID, delegatedEmail)
	return event
}

// mergeAuditMetadata folds the delegating user's identity into an event's
// Metadata without assuming the existing value is JSON — call sites write both
// JSON objects and bare "key=value" strings today, so this only ever appends.
func mergeAuditMetadata(existing, delegatedUserID, delegatedEmail string) string {
	addition := fmt.Sprintf("delegated_user_id=%s", delegatedUserID)
	if delegatedEmail != "" {
		addition = fmt.Sprintf("%s delegated_user_email=%s", addition, delegatedEmail)
	}
	if existing == "" {
		return addition
	}
	return existing + " " + addition
}

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
)

// TestApplyDelegationActor pins the audit rewrite that keeps an agent's actions
// distinguishable from its user's.
//
// RFC 8693 §1.1 draws the line this implements: delegation is "A representing
// B" with A keeping its own identity, as opposed to impersonation where A is
// indistinguishable from B. If the audit trail cannot tell them apart, an agent
// doing something damaging reads as the human doing it — and that cannot be
// fixed after the fact, because the information was never written.
func TestApplyDelegationActor(t *testing.T) {
	base := func() audit.Event {
		return audit.Event{
			Action:     constants.AuditLogoutEvent,
			ActorID:    "user-123",
			ActorType:  constants.AuditActorTypeUser,
			ActorEmail: "alice@example.com",
		}
	}

	t.Run("an ordinary caller is untouched", func(t *testing.T) {
		got := applyDelegationActor("", base())
		assert.Equal(t, base(), got,
			"a non-delegated call must produce byte-for-byte the event it always did")
	})

	t.Run("whitespace-only actor is not a delegation", func(t *testing.T) {
		assert.Equal(t, base(), applyDelegationActor("   ", base()))
	})

	t.Run("the agent becomes the actor and the user is preserved", func(t *testing.T) {
		got := applyDelegationActor("agent-client-id", base())

		assert.Equal(t, "agent-client-id", got.ActorID, "the AGENT did this")
		assert.Equal(t, constants.AuditActorTypeAgent, got.ActorType)
		assert.Empty(t, got.ActorEmail,
			"an agent has no mailbox, and leaving the user's address here is exactly the "+
				"confusion being removed")
		assert.Contains(t, got.Metadata, "delegated_user_id=user-123", "…for WHOM must survive")
		assert.Contains(t, got.Metadata, "delegated_user_email=alice@example.com")
		assert.Equal(t, constants.AuditLogoutEvent, got.Action, "nothing else is rewritten")
	})

	t.Run("existing metadata is appended to, never replaced", func(t *testing.T) {
		e := base()
		e.Metadata = `{"reason":"user_initiated"}`
		got := applyDelegationActor("agent-client-id", e)

		assert.Contains(t, got.Metadata, `{"reason":"user_initiated"}`,
			"call sites write both JSON objects and bare key=value strings, so this only ever appends")
		assert.Contains(t, got.Metadata, "delegated_user_id=user-123")
	})

	t.Run("an event with no email omits the email key", func(t *testing.T) {
		e := base()
		e.ActorEmail = ""
		got := applyDelegationActor("agent-client-id", e)

		assert.Contains(t, got.Metadata, "delegated_user_id=user-123")
		assert.NotContains(t, got.Metadata, "delegated_user_email")
	})
}

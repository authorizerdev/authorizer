package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authctx"
	"github.com/authorizerdev/authorizer/internal/authorization/engine"
)

// advStubEngine implements just enough of engine.AuthorizationEngine for the
// agent-detection path. Every other method panics so an accidental dependency
// on it is loud rather than silent.
type advStubEngine struct {
	engine.AuthorizationEngine

	modelID     string
	readModelFn func() (string, string, error)
	typeNames   []string
	typeNameErr error

	typeNamesCalls int
}

func (e *advStubEngine) ReadModel(context.Context) (string, string, error) {
	if e.readModelFn != nil {
		return e.readModelFn()
	}
	return e.modelID, "", nil
}

func (e *advStubEngine) TypeNames(context.Context) ([]string, error) {
	e.typeNamesCalls++
	if e.typeNameErr != nil {
		return nil, e.typeNameErr
	}
	return e.typeNames, nil
}

func advDelegatedCtx(userID, actorID string) context.Context {
	return authctx.WithPrincipal(context.Background(), &authctx.Principal{
		UserID:  userID,
		ActorID: actorID,
	})
}

// TestAdvAgentDetectionFailsOpen attacks internal/service/fga_agent.go:60-100.
// agentSubjectsEnabled returns false on ANY error resolving the model, and
// delegationSubjects then collapses to the single user subject — i.e. the agent
// half of the intersection disappears and the agent inherits the delegating
// user's FULL authority.
//
// The critical property being probed: this happens INDEPENDENTLY of whether the
// engine can still answer Check. ReadModel/TypeNames failing while BatchCheck
// keeps working (a transient datastore hiccup on the model read path, a model
// whose DSL rendering fails, a permissions difference on ReadAuthorizationModel)
// yields user-level access rather than a denial.
func TestAdvAgentDetectionFailsOpen(t *testing.T) {
	t.Run("TypeNames error disables the agent subject", func(t *testing.T) {
		p := &provider{}
		p.AuthzEngine = &advStubEngine{
			modelID:     "model-1",
			typeNameErr: errors.New("datastore unavailable"),
		}
		got := p.delegationSubjects(advDelegatedCtx("alice", "bot"), "user:alice")
		assert.Equal(t, []string{"agent:bot", "user:alice"}, got,
			"FAIL-OPEN: a TypeNames error drops agent:bot and leaves the agent with the user's full authority")
	})

	t.Run("ReadModel error disables the agent subject", func(t *testing.T) {
		p := &provider{}
		p.AuthzEngine = &advStubEngine{
			readModelFn: func() (string, string, error) { return "", "", errors.New("model render failed") },
		}
		got := p.delegationSubjects(advDelegatedCtx("alice", "bot"), "user:alice")
		assert.Equal(t, []string{"agent:bot", "user:alice"}, got,
			"FAIL-OPEN: a ReadModel error drops agent:bot")
	})

	t.Run("control: healthy engine with an agent type intersects", func(t *testing.T) {
		p := &provider{}
		p.AuthzEngine = &advStubEngine{modelID: "model-1", typeNames: []string{"agent", "user"}}
		got := p.delegationSubjects(advDelegatedCtx("alice", "bot"), "user:alice")
		require.Equal(t, []string{"agent:bot", "user:alice"}, got)
	})
}

// TestAdvAgentDetectionCacheIsStickyOnEmptyModelID probes the TTL branch at
// internal/service/fga_agent.go:78-80: when ReadModel returns an EMPTY model id
// the previous answer is reused for agentModelTTL regardless of what the model
// now says.
func TestAdvAgentDetectionCacheIsStickyOnEmptyModelID(t *testing.T) {
	stub := &advStubEngine{modelID: "model-1", typeNames: []string{"agent", "user"}}
	p := &provider{}
	p.AuthzEngine = stub

	ctx := advDelegatedCtx("alice", "bot")
	require.Equal(t, []string{"agent:bot", "user:alice"}, p.delegationSubjects(ctx, "user:alice"))
	require.Equal(t, 1, stub.typeNamesCalls)

	// The engine now cannot name the model (empty id) and the model no longer
	// declares `agent`. Detection must not keep serving the stale "enabled".
	stub.modelID = ""
	stub.typeNames = []string{"user"}
	got := p.delegationSubjects(ctx, "user:alice")
	assert.Equal(t, []string{"user:alice"}, got,
		"an empty model id serves the cached answer for up to %s", agentModelTTL)
	assert.Equal(t, 2, stub.typeNamesCalls, "re-detection should have run")
}

// TestAdvAgentSubjectStringIsUnvalidated pins that delegationSubjects performs
// no shape validation on ActorID, unlike machineFgaSubject (internal/service/
// fga.go:181) which rejects ":#@ \t\n" before building a subject string.
func TestAdvAgentSubjectStringIsUnvalidated(t *testing.T) {
	p := &provider{}
	p.AuthzEngine = &advStubEngine{modelID: "m", typeNames: []string{"agent", "user"}}

	for _, actor := range []string{"bot#viewer", "bot:extra", "*", "a b"} {
		got := p.delegationSubjects(advDelegatedCtx("alice", actor), "user:alice")
		assert.NotEqual(t, "agent:"+actor, got[0],
			"ActorID %q reaches the engine verbatim as an FGA subject with no shape guard", actor)
	}
}

var _ = time.Second

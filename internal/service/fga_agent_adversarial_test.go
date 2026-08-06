package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/metrics"
)

// These tests were written as ATTACKS on the agent-delegation path and each one
// originally passed against a real defect. They are kept in attack form — the
// failure message describes what an attacker gets if the assertion ever flips
// back — so a regression reads as an exploit rather than as a diff.

// advStubEngine implements just enough of engine.AuthorizationEngine for the
// agent-detection path. Every other method is nil-embedded so an accidental
// dependency on it panics loudly rather than silently returning a zero value.
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

// advDelegatedCaller is an agent acting on behalf of a user.
func advDelegatedCaller(userID, actorID string) fgaCaller {
	return fgaCaller{subject: "user:" + userID, actorID: actorID}
}

// TestAdvAgentDetectionFailsClosed covers the fail-OPEN defect this path shipped
// with: agentSubjectsEnabled swallowed every error and delegationSubjects then
// collapsed to the single user subject, so the agent half of the intersection
// disappeared and the agent inherited the delegating user's FULL authority.
//
// The property that made it exploitable: it did not require the engine to be
// down. ReadModel or TypeNames failing while Check keeps working — a hiccup on
// the model-read path, a DSL rendering failure, a narrower permission on
// ReadAuthorizationModel — was enough to silently widen every agent.
func TestAdvAgentDetectionFailsClosed(t *testing.T) {
	t.Run("TypeNames error denies", func(t *testing.T) {
		p := &provider{}
		p.AuthzEngine = &advStubEngine{
			modelID:     "model-1",
			typeNameErr: errors.New("datastore unavailable"),
		}
		got, err := p.delegationSubjects(context.Background(), advDelegatedCaller("alice", "bot"), "user:alice", metrics.FgaOpCheckPermissions)
		require.Error(t, err,
			"FAIL-OPEN: a TypeNames error must not drop agent:bot and leave the agent holding the user's full authority")
		assert.Nil(t, got)
	})

	t.Run("ReadModel error denies", func(t *testing.T) {
		p := &provider{}
		p.AuthzEngine = &advStubEngine{
			readModelFn: func() (string, string, error) { return "", "", errors.New("model render failed") },
		}
		got, err := p.delegationSubjects(context.Background(), advDelegatedCaller("alice", "bot"), "user:alice", metrics.FgaOpCheckPermissions)
		require.Error(t, err, "FAIL-OPEN: a ReadModel error must not drop agent:bot")
		assert.Nil(t, got)
	})

	t.Run("no engine denies", func(t *testing.T) {
		p := &provider{}
		_, err := p.delegationSubjects(context.Background(), advDelegatedCaller("alice", "bot"), "user:alice", metrics.FgaOpCheckPermissions)
		require.Error(t, err)
	})

	t.Run("control: healthy engine with an agent type intersects", func(t *testing.T) {
		p := &provider{}
		p.AuthzEngine = &advStubEngine{modelID: "model-1", typeNames: []string{"agent", "user"}}
		got, err := p.delegationSubjects(context.Background(), advDelegatedCaller("alice", "bot"), "user:alice", metrics.FgaOpCheckPermissions)
		require.NoError(t, err)
		require.Equal(t, []string{"agent:bot", "user:alice"}, got,
			"the agent must come first so a denied agent short-circuits the user check")
	})

	t.Run("model without an agent type authorizes the user alone", func(t *testing.T) {
		p := &provider{}
		p.AuthzEngine = &advStubEngine{modelID: "model-1", typeNames: []string{"user", "document"}}
		got, err := p.delegationSubjects(context.Background(), advDelegatedCaller("alice", "bot"), "user:alice", metrics.FgaOpCheckPermissions)
		require.NoError(t, err,
			"a model with no agent type is the documented compatibility path, not an error — "+
				"checking agent:bot against it would ERROR and deny every delegated request")
		assert.Equal(t, []string{"user:alice"}, got)
	})
}

// TestAdvAgentDetectionIsCachedPerModel pins the cache contract. Detection reads
// the model from the datastore, far too expensive per check, so the answer is
// keyed on the model id — a model write mints a new id and invalidates it for
// free.
//
// The staleness that would be dangerous is a stale "enabled=false" hiding a
// model that now declares `agent`. That is unreachable: gaining an agent type
// requires a model write, which changes the id, which misses the cache. The
// reverse (stale "enabled=true" against a model that dropped `agent`) sends
// agent:bot to a model with no such type, which ERRORS and therefore denies.
// Both directions are safe.
func TestAdvAgentDetectionIsCachedPerModel(t *testing.T) {
	stub := &advStubEngine{modelID: "model-1", typeNames: []string{"agent", "user"}}
	p := &provider{}
	p.AuthzEngine = stub
	caller := advDelegatedCaller("alice", "bot")

	got, err := p.delegationSubjects(context.Background(), caller, "user:alice", metrics.FgaOpCheckPermissions)
	require.NoError(t, err)
	require.Equal(t, []string{"agent:bot", "user:alice"}, got)
	require.Equal(t, 1, stub.typeNamesCalls)

	// Same model id: served from cache, no second datastore read.
	_, err = p.delegationSubjects(context.Background(), caller, "user:alice", metrics.FgaOpCheckPermissions)
	require.NoError(t, err)
	assert.Equal(t, 1, stub.typeNamesCalls, "a repeat check on the same model must not re-read it")

	// A model WRITE mints a new id, which must invalidate without any TTL wait.
	stub.modelID = "model-2"
	stub.typeNames = []string{"user"}
	got, err = p.delegationSubjects(context.Background(), caller, "user:alice", metrics.FgaOpCheckPermissions)
	require.NoError(t, err)
	assert.Equal(t, []string{"user:alice"}, got, "a new model id must re-detect, not serve the stale answer")
	assert.Equal(t, 2, stub.typeNamesCalls)
}

// TestAdvAgentSubjectStringIsValidated covers the missing shape guard: ActorID
// was concatenated into an OpenFGA subject verbatim, unlike machineFgaSubject
// which rejects ":#@ \t\n" first. A separator smuggled through addresses a
// DIFFERENT subject than the one authenticated — "agent:bot#member" is a
// userset, not a concrete agent.
func TestAdvAgentSubjectStringIsValidated(t *testing.T) {
	p := &provider{}
	p.AuthzEngine = &advStubEngine{modelID: "m", typeNames: []string{"agent", "user"}}

	for _, actor := range []string{"bot#viewer", "bot:extra", "a b", "bot\nx", "bot@host"} {
		t.Run(actor, func(t *testing.T) {
			got, err := p.delegationSubjects(context.Background(), advDelegatedCaller("alice", actor), "user:alice", metrics.FgaOpCheckPermissions)
			require.Error(t, err,
				"ActorID %q must not reach the engine verbatim as an FGA subject", actor)
			assert.Nil(t, got)
		})
	}
}

// TestAdvDelegatedCallerCannotNameAnotherSubject covers the one-parameter defeat
// of the whole intersection.
//
// resolveFgaSubject honours SELF-specification for any caller (it is exactly
// what the token proves), and CheckPermissions used to skip the delegation
// expansion whenever `user` was supplied. A delegated agent could therefore echo
// back its own subject and be authorized as the user ALONE — the agent half
// silently dropped by a field the attacker controls. The gate now keys on who
// the caller is, never on what they typed.
func TestAdvDelegatedCallerCannotNameAnotherSubject(t *testing.T) {
	p := &provider{}
	p.AuthzEngine = &advStubEngine{modelID: "m", typeNames: []string{"agent", "user"}}
	caller := advDelegatedCaller("alice", "bot")
	ctx := context.Background()

	t.Run("self-specification still intersects", func(t *testing.T) {
		subject, err := p.resolveFgaSubject(ctx, RequestMetadata{}, caller, "user:alice")
		require.NoError(t, err, "self-specification stays accepted; it is what the token proves")

		got, err := p.delegationSubjects(ctx, caller, subject, metrics.FgaOpCheckPermissions)
		require.NoError(t, err)
		assert.Equal(t, []string{"agent:bot", "user:alice"}, got,
			"echoing back your own subject must NOT shed the agent half")
	})

	t.Run("bare id normalizes to self and still intersects", func(t *testing.T) {
		subject, err := p.resolveFgaSubject(ctx, RequestMetadata{}, caller, "alice")
		require.NoError(t, err)

		got, err := p.delegationSubjects(ctx, caller, subject, metrics.FgaOpCheckPermissions)
		require.NoError(t, err)
		assert.Equal(t, []string{"agent:bot", "user:alice"}, got)
	})

	t.Run("another subject is refused", func(t *testing.T) {
		_, err := p.resolveFgaSubject(ctx, RequestMetadata{}, caller, "user:bob")
		require.Error(t, err, "a delegated token must never widen its subject")
	})

	t.Run("a non-delegated caller is unaffected", func(t *testing.T) {
		ordinary := fgaCaller{subject: "user:alice"}
		subject, err := p.resolveFgaSubject(ctx, RequestMetadata{}, ordinary, "user:alice")
		require.NoError(t, err)

		got, err := p.delegationSubjects(ctx, ordinary, subject, metrics.FgaOpCheckPermissions)
		require.NoError(t, err)
		assert.Equal(t, []string{"user:alice"}, got,
			"an ordinary caller must see byte-for-byte the pre-delegation behaviour")
	})
}

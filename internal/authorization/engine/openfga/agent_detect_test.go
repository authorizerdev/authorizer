package openfga

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTypeNamesSeesSubjectOnlyTypes pins the property TypeRelations cannot
// provide: a type declaring no relations must still be reported.
//
// The canonical agent model declares `type agent` with no relations, because an
// agent is only ever a SUBJECT, never the object of a permission. TypeRelations
// omits such types, so agent-subject detection built on it would silently never
// activate — the feature would look wired up and do nothing.
func TestTypeNamesSeesSubjectOnlyTypes(t *testing.T) {
	ctx := context.Background()
	eng, _ := newTestEngine(t)

	_, err := eng.WriteModel(ctx, `
model
  schema 1.1

type user

type agent

type document
  relations
    define viewer: [user, agent]
    define can_view: viewer
`)
	require.NoError(t, err)

	names, err := eng.TypeNames(ctx)
	require.NoError(t, err)
	assert.Contains(t, names, "agent", "a relation-less subject type must be visible")
	assert.Contains(t, names, "user")
	assert.Contains(t, names, "document")

	// Demonstrates why TypeNames had to be added at all.
	rels, err := eng.TypeRelations(ctx)
	require.NoError(t, err)
	_, present := rels["agent"]
	assert.False(t, present, "TypeRelations omits relation-less types — this is the gap TypeNames fills")
}

// TestTypeNamesWithoutModel pins the no-model contract.
func TestTypeNamesWithoutModel(t *testing.T) {
	eng, _ := newTestEngine(t)
	_, err := eng.TypeNames(context.Background())
	require.Error(t, err, "no model must be an error, not an empty list that reads as 'no agent type'")
}

// TestModelRefreshAcrossReplicas is the regression test for divergent
// authorization in a multi-replica fleet.
//
// Check pins an explicit AuthorizationModelId and WriteModel only updates the
// modelID of the replica that served it, so a model written on replica A was
// never picked up by replica B until B restarted. Two replicas could evaluate
// the same request against different models indefinitely — and any behaviour
// derived from the model (agent-subject detection) would diverge with it.
//
// Both "replicas" here share one datastore, which is exactly the production
// shape.
func TestModelRefreshAcrossReplicas(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	url := "file:" + dir + "/fga.db?_pragma=busy_timeout(5000)"

	newReplica := func() *engineImpl {
		log := zerologNop()
		eng, err := New(&Config{
			Store: StoreSQLite, StoreURL: url, StoreName: "replica-test", RunMigrations: true,
		}, &Dependencies{Log: &log})
		require.NoError(t, err)
		impl, ok := eng.(*engineImpl)
		require.True(t, ok)
		t.Cleanup(impl.Close)
		return impl
	}

	replicaA := newReplica()
	replicaB := newReplica()

	// A writes the first model; B adopts it at boot.
	_, err := replicaA.WriteModel(ctx, `
model
  schema 1.1
type user
type document
  relations
    define viewer: [user]
    define can_view: viewer
`)
	require.NoError(t, err)

	// B has not seen a model written after its own boot yet. Force its next
	// ids() call to reconcile rather than waiting out the interval.
	replicaB.mu.Lock()
	replicaB.modelCheckedAt = time.Time{}
	replicaB.mu.Unlock()

	// A writes a SECOND model introducing the agent type.
	newID, err := replicaA.WriteModel(ctx, `
model
  schema 1.1
type user
type agent
type document
  relations
    define viewer: [user, agent]
    define can_view: viewer
`)
	require.NoError(t, err)

	replicaB.mu.Lock()
	replicaB.modelCheckedAt = time.Time{}
	replicaB.mu.Unlock()

	names, err := replicaB.TypeNames(ctx)
	require.NoError(t, err)
	assert.Contains(t, names, "agent",
		"replica B must adopt a model written by replica A without a restart — "+
			"otherwise agent detection is on in one replica and off in another")

	_, bModel := replicaB.ids()
	assert.Equal(t, newID, bModel, "replica B must converge on the newest model id")
}

// TestPinnedModelIsNeverRefreshed pins the operator-override contract: an
// explicit Config.ModelID means "use exactly this version", so the refresh must
// not silently move off it.
func TestPinnedModelIsNeverRefreshed(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	url := "file:" + dir + "/fga.db?_pragma=busy_timeout(5000)"

	log := zerologNop()
	seed, err := New(&Config{Store: StoreSQLite, StoreURL: url, StoreName: "pin-test", RunMigrations: true}, &Dependencies{Log: &log})
	require.NoError(t, err)
	firstID, err := seed.WriteModel(ctx, "model\n  schema 1.1\ntype user\n")
	require.NoError(t, err)
	seed.(*engineImpl).Close()

	pinnedEng, err := New(&Config{
		Store: StoreSQLite, StoreURL: url, StoreName: "pin-test", ModelID: firstID,
	}, &Dependencies{Log: &log})
	require.NoError(t, err)
	pinned := pinnedEng.(*engineImpl)
	t.Cleanup(pinned.Close)

	writer, err := New(&Config{Store: StoreSQLite, StoreURL: url, StoreName: "pin-test"}, &Dependencies{Log: &log})
	require.NoError(t, err)
	defer writer.(*engineImpl).Close()
	_, err = writer.WriteModel(ctx, "model\n  schema 1.1\ntype user\ntype agent\n")
	require.NoError(t, err)

	pinned.mu.Lock()
	pinned.modelCheckedAt = time.Time{}
	pinned.mu.Unlock()

	_, got := pinned.ids()
	assert.Equal(t, firstID, got, "a pinned model must never be refreshed away from")
}

func zerologNop() zerolog.Logger { return zerolog.Nop() }

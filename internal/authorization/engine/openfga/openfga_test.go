package openfga

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
)

// testModel mirrors the Phase 0 spike: a document with a viewer grant, a
// blocked exception, and an effective can_view = viewer but not blocked.
const testModel = `model
  schema 1.1
type user
type document
  relations
    define viewer: [user]
    define blocked: [user]
    define can_view: viewer but not blocked`

// newTestEngine constructs the embedded OpenFGA engine over the in-memory
// datastore and returns it along with its concrete type for Close().
func newTestEngine(t *testing.T) (engine.AuthorizationEngine, *engineImpl) {
	t.Helper()
	log := zerolog.New(os.Stderr)
	eng, err := New(&Config{Store: StoreMemory}, &Dependencies{Log: &log})
	require.NoError(t, err)
	require.NotNil(t, eng)

	impl, ok := eng.(*engineImpl)
	require.True(t, ok, "expected *engineImpl")
	t.Cleanup(impl.Close)
	return eng, impl
}

func TestOpenFGAEngine_MemoryStore_CheckAndListObjects(t *testing.T) {
	ctx := context.Background()
	eng, impl := newTestEngine(t)

	// A store must have been bootstrapped automatically.
	assert.NotEmpty(t, impl.StoreID(), "store ID should be bootstrapped")

	// Write the model.
	modelID, err := eng.WriteModel(ctx, testModel)
	require.NoError(t, err)
	assert.NotEmpty(t, modelID)
	assert.Equal(t, modelID, impl.ModelID())

	// Write tuples: alice is a viewer; erin is a viewer but also blocked.
	err = eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:alice", Relation: "viewer", Object: "document:1"},
		{User: "user:erin", Relation: "viewer", Object: "document:1"},
		{User: "user:erin", Relation: "blocked", Object: "document:1"},
	})
	require.NoError(t, err)

	t.Run("can_view alice is allowed", func(t *testing.T) {
		allowed, err := eng.Check(ctx, "user:alice", "can_view", "document:1")
		require.NoError(t, err)
		assert.True(t, allowed, "alice is a viewer and not blocked")
	})

	t.Run("can_view erin is denied by blocked exclusion", func(t *testing.T) {
		allowed, err := eng.Check(ctx, "user:erin", "can_view", "document:1")
		require.NoError(t, err)
		assert.False(t, allowed, "erin is blocked despite being a viewer")
	})

	t.Run("can_view unknown user is denied", func(t *testing.T) {
		allowed, err := eng.Check(ctx, "user:bob", "can_view", "document:1")
		require.NoError(t, err)
		assert.False(t, allowed, "bob has no grant")
	})

	t.Run("ListObjects returns only document:1 for alice", func(t *testing.T) {
		objects, err := eng.ListObjects(ctx, "user:alice", "can_view", "document")
		require.NoError(t, err)
		assert.Equal(t, []string{"document:1"}, objects)
	})

	t.Run("ListObjects returns nothing for blocked erin", func(t *testing.T) {
		objects, err := eng.ListObjects(ctx, "user:erin", "can_view", "document")
		require.NoError(t, err)
		assert.Empty(t, objects)
	})
}

func TestOpenFGAEngine_BatchCheck(t *testing.T) {
	ctx := context.Background()
	eng, _ := newTestEngine(t)

	_, err := eng.WriteModel(ctx, testModel)
	require.NoError(t, err)
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:alice", Relation: "viewer", Object: "document:1"},
		{User: "user:erin", Relation: "viewer", Object: "document:1"},
		{User: "user:erin", Relation: "blocked", Object: "document:1"},
	}))

	results, err := eng.BatchCheck(ctx, []engine.CheckRequest{
		{User: "user:alice", Relation: "can_view", Object: "document:1"},
		{User: "user:erin", Relation: "can_view", Object: "document:1"},
		{User: "user:bob", Relation: "can_view", Object: "document:1"},
	})
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.True(t, results[0].Allowed, "alice allowed")
	assert.False(t, results[1].Allowed, "erin blocked")
	assert.False(t, results[2].Allowed, "bob no grant")
}

func TestOpenFGAEngine_ReadWriteDeleteTuples(t *testing.T) {
	ctx := context.Background()
	eng, _ := newTestEngine(t)

	_, err := eng.WriteModel(ctx, testModel)
	require.NoError(t, err)

	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:alice", Relation: "viewer", Object: "document:1"},
	}))

	// Read back the tuple.
	res, err := eng.ReadTuples(ctx, engine.ReadTuplesFilter{Object: "document:1"})
	require.NoError(t, err)
	require.Len(t, res.Tuples, 1)
	assert.Equal(t, "user:alice", res.Tuples[0].User)
	assert.Equal(t, "viewer", res.Tuples[0].Relation)
	assert.Equal(t, "document:1", res.Tuples[0].Object)

	// Delete it and confirm it is gone.
	require.NoError(t, eng.DeleteTuples(ctx, []engine.TupleKey{
		{User: "user:alice", Relation: "viewer", Object: "document:1"},
	}))
	res, err = eng.ReadTuples(ctx, engine.ReadTuplesFilter{Object: "document:1"})
	require.NoError(t, err)
	assert.Empty(t, res.Tuples)
}

func TestOpenFGAEngine_ReadModelRoundtrip(t *testing.T) {
	ctx := context.Background()
	eng, _ := newTestEngine(t)

	// A fresh store has no model yet: ReadModel must report the typed sentinel
	// so callers can treat it as an empty state, not a failure.
	_, _, err := eng.ReadModel(ctx)
	assert.ErrorIs(t, err, engine.ErrNoModel, "ReadModel on a fresh store must return ErrNoModel")

	_, err = eng.WriteModel(ctx, testModel)
	require.NoError(t, err)

	id, dsl, err := eng.ReadModel(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, id, "ReadModel must return the active model id")
	assert.Contains(t, dsl, "type document")
	assert.Contains(t, dsl, "can_view")
}

func TestOpenFGAEngine_Reset(t *testing.T) {
	ctx := context.Background()
	eng, impl := newTestEngine(t)

	_, err := eng.WriteModel(ctx, testModel)
	require.NoError(t, err)
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "user:alice", Relation: "viewer", Object: "document:1"},
	}))

	storeBefore := impl.StoreID()
	require.NotEmpty(t, storeBefore)

	// Reset wipes the store: a fresh store ID, no active model, no tuples.
	require.NoError(t, eng.Reset(ctx))
	assert.NotEqual(t, storeBefore, impl.StoreID(), "Reset must create a new store")
	assert.Empty(t, impl.ModelID(), "Reset must clear the active model")

	// No model after reset → ReadModel reports the no-model sentinel.
	_, _, err = eng.ReadModel(ctx)
	assert.ErrorIs(t, err, engine.ErrNoModel, "ReadModel after reset must return ErrNoModel")

	// The engine is reusable: a new model and tuples can be written.
	_, err = eng.WriteModel(ctx, testModel)
	require.NoError(t, err)
	res, err := eng.ReadTuples(ctx, engine.ReadTuplesFilter{})
	require.NoError(t, err)
	assert.Empty(t, res.Tuples, "tuples from before the reset must be gone")
}

func TestOpenFGAEngine_CheckBeforeModelFailsClosed(t *testing.T) {
	ctx := context.Background()
	eng, _ := newTestEngine(t)

	allowed, err := eng.Check(ctx, "user:alice", "can_view", "document:1")
	assert.Error(t, err, "checking before a model is written must error")
	assert.False(t, allowed, "must fail closed")
}

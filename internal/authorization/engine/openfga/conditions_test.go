package openfga

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/authorization/engine"
)

// conditionModel is the time-limited agent grant template documented in the
// FGA guide: an agent's access to a tool expires, either granted directly or
// through a task the agent is assigned to.
const conditionModel = `model
  schema 1.1
type user
type agent
type task
  relations
    define assignee: [agent]
type tool
  relations
    define can_call: [user, agent with non_expired_grant, task#assignee with non_expired_grant]
condition non_expired_grant(current_time: timestamp, grant_time: timestamp, grant_duration: duration) {
  current_time < grant_time + grant_duration
}`

func grant(start time.Time, d string) *engine.TupleCondition {
	return &engine.TupleCondition{
		Name:    "non_expired_grant",
		Context: map[string]any{"grant_time": start.UTC().Format(time.RFC3339), "grant_duration": d},
	}
}

func TestConditions_TimeLimitedAgentGrants(t *testing.T) {
	ctx := context.Background()
	eng, _ := newTestEngine(t)
	_, err := eng.WriteModel(ctx, conditionModel)
	require.NoError(t, err)

	now := time.Now()
	require.NoError(t, eng.WriteTuples(ctx, []engine.TupleKey{
		{User: "agent:fresh", Relation: "can_call", Object: "tool:search", Condition: grant(now, "10m")},
		{User: "agent:stale", Relation: "can_call", Object: "tool:search", Condition: grant(now.Add(-time.Hour), "10m")},
		{User: "agent:worker", Relation: "assignee", Object: "task:t1"},
		{User: "task:t1#assignee", Relation: "can_call", Object: "tool:deploy", Condition: grant(now, "10m")},
		{User: "user:alice", Relation: "can_call", Object: "tool:search"},
	}))

	t.Run("Check uses the server clock", func(t *testing.T) {
		ok, err := eng.Check(ctx, "agent:fresh", "can_call", "tool:search")
		require.NoError(t, err, "a conditional tuple must not error: the engine supplies current_time")
		assert.True(t, ok)
		ok, err = eng.Check(ctx, "agent:stale", "can_call", "tool:search")
		require.NoError(t, err)
		assert.False(t, ok, "a grant older than its duration must deny")
	})

	t.Run("task-scoped grant reaches the assigned agent only", func(t *testing.T) {
		ok, err := eng.Check(ctx, "agent:worker", "can_call", "tool:deploy")
		require.NoError(t, err)
		assert.True(t, ok)
		ok, err = eng.Check(ctx, "agent:fresh", "can_call", "tool:deploy")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("unconditional tuples are unaffected", func(t *testing.T) {
		ok, err := eng.Check(ctx, "user:alice", "can_call", "tool:search")
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("a caller-supplied current_time cannot revive an expired grant", func(t *testing.T) {
		res, err := eng.BatchCheck(ctx, []engine.CheckRequest{{
			User: "agent:stale", Relation: "can_call", Object: "tool:search",
			Context: map[string]any{engine.ContextKeyCurrentTime: now.Add(-2 * time.Hour).UTC().Format(time.RFC3339)},
		}})
		require.NoError(t, err)
		require.Len(t, res, 1)
		assert.False(t, res[0].Allowed, "current_time is server-owned; the engine overwrites it")
	})

	t.Run("ListObjects and ListUsers evaluate conditions instead of erroring", func(t *testing.T) {
		objs, err := eng.ListObjects(ctx, "agent:fresh", "can_call", "tool")
		require.NoError(t, err)
		assert.Equal(t, []string{"tool:search"}, objs)
		objs, err = eng.ListObjects(ctx, "agent:stale", "can_call", "tool")
		require.NoError(t, err)
		assert.Empty(t, objs)
		users, err := eng.ListUsers(ctx, "tool:search", "can_call", "agent")
		require.NoError(t, err)
		assert.Equal(t, []string{"agent:fresh"}, users)
	})

	t.Run("ReadTuples returns the stored condition", func(t *testing.T) {
		res, err := eng.ReadTuples(ctx, engine.ReadTuplesFilter{User: "agent:fresh", Relation: "can_call", Object: "tool:search"})
		require.NoError(t, err)
		require.Len(t, res.Tuples, 1)
		c := res.Tuples[0].Condition
		require.NotNil(t, c)
		assert.Equal(t, "non_expired_grant", c.Name)
		assert.Equal(t, "10m", c.Context["grant_duration"])

		res, err = eng.ReadTuples(ctx, engine.ReadTuplesFilter{User: "user:alice", Relation: "can_call", Object: "tool:search"})
		require.NoError(t, err)
		require.Len(t, res.Tuples, 1)
		assert.Nil(t, res.Tuples[0].Condition)
	})

	t.Run("revoking deletes by key, without the condition", func(t *testing.T) {
		require.NoError(t, eng.DeleteTuples(ctx, []engine.TupleKey{{User: "agent:fresh", Relation: "can_call", Object: "tool:search"}}))
		ok, err := eng.Check(ctx, "agent:fresh", "can_call", "tool:search")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("the engine itself refuses a stored current_time", func(t *testing.T) {
		c := grant(now.Add(-time.Hour), "10m")
		c.Context[engine.ContextKeyCurrentTime] = "2000-01-01T00:00:00Z"
		err := eng.WriteTuples(ctx, []engine.TupleKey{{User: "agent:frozen", Relation: "can_call", Object: "tool:search", Condition: c}})
		require.Error(t, err, "a stored current_time would override the server clock; every write path must refuse it")
		ok, cErr := eng.Check(ctx, "agent:frozen", "can_call", "tool:search")
		require.NoError(t, cErr)
		assert.False(t, ok)
	})

	t.Run("an undeclared condition is rejected at write", func(t *testing.T) {
		err := eng.WriteTuples(ctx, []engine.TupleKey{{User: "agent:x", Relation: "can_call", Object: "tool:search",
			Condition: &engine.TupleCondition{Name: "no_such_condition"}}})
		assert.Error(t, err)
	})
}

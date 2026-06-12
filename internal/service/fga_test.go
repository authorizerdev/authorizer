package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestToContextualTuples covers the public-boundary validation for
// client-supplied contextual tuples: empty input, field presence, and the
// per-check count cap that must hold regardless of the embedded engine's own
// limits.
func TestToContextualTuples(t *testing.T) {
	t.Run("nil input returns nil without error", func(t *testing.T) {
		out, err := toContextualTuples(nil)
		require.NoError(t, err)
		assert.Nil(t, out)
	})

	t.Run("valid tuples convert one to one", func(t *testing.T) {
		in := []*model.FgaTupleInput{
			{User: "user:1", Relation: "viewer", Object: "document:a"},
			{User: "user:2", Relation: "editor", Object: "document:b"},
		}
		out, err := toContextualTuples(in)
		require.NoError(t, err)
		require.Len(t, out, 2)
		assert.Equal(t, "user:1", out[0].User)
		assert.Equal(t, "editor", out[1].Relation)
	})

	t.Run("blank field is rejected", func(t *testing.T) {
		in := []*model.FgaTupleInput{{User: "user:1", Relation: " ", Object: "document:a"}}
		_, err := toContextualTuples(in)
		require.Error(t, err)
	})

	t.Run("count above the cap is rejected", func(t *testing.T) {
		in := make([]*model.FgaTupleInput, maxContextualTuplesPerCheck+1)
		for i := range in {
			in[i] = &model.FgaTupleInput{User: "user:1", Relation: "viewer", Object: "document:a"}
		}
		_, err := toContextualTuples(in)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many contextual tuples")
	})

	t.Run("count at the cap is accepted", func(t *testing.T) {
		in := make([]*model.FgaTupleInput, maxContextualTuplesPerCheck)
		for i := range in {
			in[i] = &model.FgaTupleInput{User: "user:1", Relation: "viewer", Object: "document:a"}
		}
		out, err := toContextualTuples(in)
		require.NoError(t, err)
		assert.Len(t, out, maxContextualTuplesPerCheck)
	})
}

package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// The gRPC/REST transports must carry FGA conditions both ways, and must pass a
// condition on a contextual tuple through so the service layer can reject it
// explicitly instead of the transport dropping it silently.
func TestFgaConditionMapping(t *testing.T) {
	ctxStruct, err := structpb.NewStruct(map[string]any{"grant_time": "2026-11-27T10:00:00Z", "grant_duration": "10m"})
	require.NoError(t, err)
	cond := &authorizerv1.FgaCondition{Name: "non_expired_grant", Context: ctxStruct}

	t.Run("tuple write input carries the condition", func(t *testing.T) {
		out := modelFgaTupleInputs([]*authorizerv1.FgaTupleInput{
			{User: "agent:a", Relation: "viewer", Object: "document:1", Condition: cond},
			{User: "user:u", Relation: "viewer", Object: "document:1"},
		})
		require.Len(t, out, 2)
		require.NotNil(t, out[0].Condition)
		assert.Equal(t, "non_expired_grant", out[0].Condition.Name)
		assert.Equal(t, "10m", out[0].Condition.Context["grant_duration"])
		assert.Nil(t, out[1].Condition)
	})

	t.Run("contextual tuple condition reaches the service to be rejected", func(t *testing.T) {
		out := protoToModelPermissionChecks([]*authorizerv1.PermissionCheckInput{{
			Relation: "viewer", Object: "document:1",
			ContextualTuples: []*authorizerv1.FgaTupleInput{{User: "user:u", Relation: "viewer", Object: "document:1", Condition: cond}},
		}})
		require.Len(t, out, 1)
		require.Len(t, out[0].ContextualTuples, 1)
		assert.NotNil(t, out[0].ContextualTuples[0].Condition)
	})

	t.Run("read response projects the condition", func(t *testing.T) {
		res := projectFgaTuples(&model.FgaTuples{Tuples: []*model.FgaTuple{
			{User: "agent:a", Relation: "viewer", Object: "document:1",
				Condition: &model.FgaCondition{Name: "non_expired_grant", Context: map[string]any{"grant_duration": "10m"}}},
			{User: "user:u", Relation: "viewer", Object: "document:1"},
		}})
		require.Len(t, res.Tuples, 2)
		require.NotNil(t, res.Tuples[0].Condition)
		assert.Equal(t, "non_expired_grant", res.Tuples[0].Condition.Name)
		assert.Equal(t, "10m", res.Tuples[0].Condition.Context.AsMap()["grant_duration"])
		assert.Nil(t, res.Tuples[1].Condition)
	})
}

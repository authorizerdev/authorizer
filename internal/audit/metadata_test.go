package audit

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMetadataWithProtocol covers the four shapes metadataWithProtocol must
// handle: no protocol, empty metadata, JSON-object metadata, and opaque
// (non-JSON) metadata.
func TestMetadataWithProtocol(t *testing.T) {
	t.Run("no protocol returns metadata unchanged", func(t *testing.T) {
		require.Equal(t, "", metadataWithProtocol("", ""))
		require.Equal(t, "existing", metadataWithProtocol("existing", ""))
	})

	t.Run("empty metadata yields protocol-only json", func(t *testing.T) {
		got := metadataWithProtocol("", "rest")
		require.JSONEq(t, `{"protocol":"rest"}`, got)
	})

	t.Run("json-object metadata gains protocol key", func(t *testing.T) {
		got := metadataWithProtocol(`{"foo":"bar"}`, "grpc")
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(got), &m))
		require.Equal(t, "grpc", m["protocol"])
		require.Equal(t, "bar", m["foo"])
	})

	t.Run("opaque metadata preserved under metadata key", func(t *testing.T) {
		got := metadataWithProtocol("plain text", "graphql")
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(got), &m))
		require.Equal(t, "graphql", m["protocol"])
		require.Equal(t, "plain text", m["metadata"])
	})
}

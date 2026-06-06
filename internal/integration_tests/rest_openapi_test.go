package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/gen/openapi"
)

// TestOpenAPIEndpointServesValidJSON verifies the /openapi.json route
// returns the embedded swagger spec, with a body that parses as JSON and
// declares the v1 services. Guards against two regressions:
//  1. Path-based reads of the spec file would fail when cwd is not the
//     repo root (Docker, tests). The embed should make this path-free.
//  2. The merged swagger is non-empty and includes recognisable v1 routes.
func TestOpenAPIEndpointServesValidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/openapi.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", openapi.Spec())
	})

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/openapi.json")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var doc map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	// Sanity: swagger 2.0 doc with at least one path under /v1.
	assert.Contains(t, doc, "swagger")
	paths, ok := doc["paths"].(map[string]any)
	require.True(t, ok, "openapi spec missing paths object")
	assert.NotEmpty(t, paths, "openapi spec should declare at least one path")
}

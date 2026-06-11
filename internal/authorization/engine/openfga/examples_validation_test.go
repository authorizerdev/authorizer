package openfga

import (
	"context"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests are the in-repo equivalent of `fga model validate` (the
// openfga/agent-skills "always validate models" workflow step): every DSL we
// ship — the dashboard example catalog and the model-editor placeholder — is
// run through the real embedded engine, so a malformed example can never
// reach users.

// tsDslRe matches TypeScript template literals holding an OpenFGA model. The
// DSL never contains a backtick, so a lazy match to the closing backtick is
// safe.
var tsDslRe = regexp.MustCompile("(?s)`model\n  schema 1\\.1.*?`")

// validateAll writes each extracted DSL to a fresh embedded engine and fails
// with the engine's own error message on the first invalid one.
func validateAll(t *testing.T, source string, dsls []string) {
	t.Helper()
	ctx := context.Background()
	eng, _ := newTestEngine(t)
	for i, dsl := range dsls {
		_, err := eng.WriteModel(ctx, dsl)
		assert.NoError(t, err, "%s: model #%d is not valid OpenFGA DSL:\n%s", source, i+1, dsl)
	}
}

func TestDashboardModelExamplesAreValidOpenFGADSL(t *testing.T) {
	const path = "../../../../web/dashboard/src/pages/authorization/modelDsl.ts"
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	matches := tsDslRe.FindAllString(string(content), -1)
	// The catalog ships 11 examples; a drop below that means the extraction
	// regex broke or examples were removed — both worth failing loudly on.
	require.GreaterOrEqual(t, len(matches), 11, "expected the full example catalog in %s", path)

	dsls := make([]string, 0, len(matches))
	for _, m := range matches {
		dsls = append(dsls, m[1:len(m)-1]) // strip the backticks
	}
	validateAll(t, "modelDsl.ts", dsls)
}

func TestModelEditorPlaceholderIsValidOpenFGADSL(t *testing.T) {
	const path = "../../../../web/dashboard/src/pages/authorization/Model.tsx"
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	matches := tsDslRe.FindAllString(string(content), -1)
	require.NotEmpty(t, matches, "expected the PLACEHOLDER model in %s", path)

	dsls := make([]string, 0, len(matches))
	for _, m := range matches {
		dsls = append(dsls, m[1:len(m)-1])
	}
	validateAll(t, "Model.tsx", dsls)
}

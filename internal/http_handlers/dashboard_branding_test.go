package http_handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
)

// getDashboard renders the real dashboard.tmpl with the production `json`
// FuncMap and returns the window.__authorizer__ payload it injects.
func getDashboard(t *testing.T, cfg *config.Config) (map[string]any, string) {
	t.Helper()
	logger := zerolog.Nop()
	h := &httpProvider{Config: cfg, Dependencies: Dependencies{Log: &logger}}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// The exact FuncMap the production router uses (internal/server).
	router.SetFuncMap(TemplateFuncMap())
	router.LoadHTMLFiles("../../web/templates/dashboard.tmpl")
	router.GET("/dashboard", h.DashboardHandler())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	m := regexp.MustCompile(`window\.__authorizer__ = (.*);`).FindStringSubmatch(body)
	require.Len(t, m, 2, "dashboard shell must inject window.__authorizer__")
	var data map[string]any
	require.NoError(t, json.Unmarshal([]byte(m[1]), &data))
	return data, body
}

// TestDashboardHandlerInjectsOrganizationBranding guards issue #444: the admin
// login page and sidebar must show the deployment's --organization-name /
// --organization-logo (as the /app login UI already does), not a hard-coded
// Authorizer logo.
func TestDashboardHandlerInjectsOrganizationBranding(t *testing.T) {
	data, _ := getDashboard(t, &config.Config{
		OrganizationName: "Acme Corp",
		OrganizationLogo: "https://acme.example/logo.png",
	})
	assert.Equal(t, true, data["isOnboardingCompleted"])
	assert.Equal(t, "Acme Corp", data["organizationName"])
	assert.Equal(t, "https://acme.example/logo.png", data["organizationLogo"])
}

// The values come from operator flags, but they are rendered inside an inline
// <script>; a hostile name must not terminate the script element or open an
// HTML comment in the rendered page.
func TestDashboardHandlerBrandingCannotBreakOutOfScript(t *testing.T) {
	const hostile = `Evil</script><script>alert(1)</script><!--`
	data, body := getDashboard(t, &config.Config{
		OrganizationName: hostile,
		OrganizationLogo: "https://acme.example/logo.png",
	})
	assert.Equal(t, hostile, data["organizationName"], "value must round-trip intact")
	assert.NotContains(t, body, "</script><script>alert(1)")
	assert.NotContains(t, body, "alert(1)</script>")
	assert.NotContains(t, body, "<!--", "no HTML comment may be opened by injected data")
}

// TestTemplateJSONFuncNeverEmitsRawAngleBrackets pins the production json
// template func's contract directly: whatever the input, its output contains
// no raw "<", so it can't close a <script> or open a comment.
func TestTemplateJSONFuncNeverEmitsRawAngleBrackets(t *testing.T) {
	jsonFn, ok := TemplateFuncMap()["json"].(func(interface{}) template.JS)
	require.True(t, ok, "json template func has an unexpected signature")
	for _, in := range []any{
		"</script>", "<!--", "<script>alert(1)</script>",
		map[string]any{"k": "</ScRiPt>"}, []string{"<", ">", "&"},
	} {
		out := string(jsonFn(in))
		assert.NotContains(t, out, "<", "input %v", in)
		var back any
		require.NoError(t, json.Unmarshal([]byte(out), &back), "output must stay valid JSON for %v", in)
	}
}

package http_handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
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
	router.SetFuncMap(template.FuncMap{
		"json": func(v any) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(strings.ReplaceAll(string(a), "</", `<\/`))
		},
	})
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
// <script>; a name containing "</script>" must not terminate it.
func TestDashboardHandlerBrandingCannotBreakOutOfScript(t *testing.T) {
	data, body := getDashboard(t, &config.Config{
		OrganizationName: `Evil</script><script>alert(1)</script>`,
		OrganizationLogo: "https://acme.example/logo.png",
	})
	assert.Equal(t, `Evil</script><script>alert(1)</script>`, data["organizationName"])
	assert.NotContains(t, body, "<script>alert(1)")
}

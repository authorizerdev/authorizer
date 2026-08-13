package http_handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	inmemorystore "github.com/authorizerdev/authorizer/internal/memory_store/in_memory"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// newAppTestProvider builds an /app handler backed by the same client stub the
// authorize tests use, with a DELIBERATELY narrow AllowedOrigins so the
// allow-list and the client's registered URIs cannot be confused for each other.
func newAppTestProvider(t *testing.T, client *schemas.Client) *httpProvider {
	t.Helper()
	logger := zerolog.Nop()
	cfg := &config.Config{
		AllowedOrigins:  []string{"https://console.example.com"},
		EnableLoginPage: true,
	}
	ms, err := inmemorystore.NewInMemoryProvider(cfg, &inmemorystore.Dependencies{Log: &logger})
	require.NoError(t, err)
	return &httpProvider{
		Config: cfg,
		Dependencies: Dependencies{
			Log:                 &logger,
			StorageProvider:     &redirectURIClientStore{client: client},
			MemoryStoreProvider: ms,
		},
	}
}

func getApp(h *httpProvider, clientID, redirectURI string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	router := gin.New()
	// The real template, so a success is a real render rather than a 200 from a
	// stub. app.tmpl uses the `json` FuncMap the production router registers, and
	// SetFuncMap must precede LoadHTMLFiles or parsing fails on the unknown
	// function.
	router.SetFuncMap(template.FuncMap{
		"json": func(v any) template.JS {
			a, _ := json.Marshal(v)
			return template.JS(strings.ReplaceAll(string(a), "</", `<\/`))
		},
	})
	router.LoadHTMLFiles("../../web/templates/app.tmpl")
	router.GET("/app", h.AppHandler())
	q := url.Values{}
	if clientID != "" {
		q.Set("client_id", clientID)
	}
	q.Set("redirect_uri", redirectURI)
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app?"+q.Encode(), nil))
	return rec
}

// TestAppHandlerHonoursRegisteredRedirectURIs is the regression guard for a bug
// that made the login page unreachable for any client whose registered
// redirect_uri was not ALSO a globally allowed origin.
//
// /authorize validates the redirect against the client and then hands the same
// value to /app to render the login page. /app validated it against
// AllowedOrigins alone, so the two halves of one flow disagreed: the request
// passed the first check and was refused by the second with "invalid redirect
// url", before the user could type anything.
//
// It stayed hidden because every fixture allow-listed its own callback origin.
// It cannot be hidden that way in production — an MCP client binds an EPHEMERAL
// loopback port, so there is no origin to allow-list in advance. Found by
// e2e-playground/tests/dcr.spec.ts; kept here so `make test` catches a
// regression without needing Docker.
func TestAppHandlerHonoursRegisteredRedirectURIs(t *testing.T) {
	const clientID = "dcr-client"
	const loopback = "http://127.0.0.1:53119/callback"

	registered := &schemas.Client{
		ID:           "id-1",
		ClientID:     clientID,
		Kind:         constants.ClientKindDynamic,
		RedirectURIs: loopback,
		IsActive:     true,
	}

	t.Run("a registered redirect outside AllowedOrigins is accepted", func(t *testing.T) {
		h := newAppTestProvider(t, registered)
		rec := getApp(h, clientID, loopback)
		assert.Equal(t, http.StatusOK, rec.Code,
			"the login page must render for a client's own registered redirect_uri: %s", rec.Body.String())
	})

	t.Run("an ephemeral port on the same loopback host is accepted", func(t *testing.T) {
		// RFC 8252 §7.3: the port of a loopback redirect is not fixed, so the
		// match must ignore it. This is the case no operator could allow-list.
		h := newAppTestProvider(t, registered)
		rec := getApp(h, clientID, "http://127.0.0.1:61234/callback")
		assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	})

	t.Run("a redirect the client did not register is still refused", func(t *testing.T) {
		// The fix must not degrade into "any redirect is fine once a client_id
		// is present" — that would be an open redirect from the login page.
		h := newAppTestProvider(t, registered)
		rec := getApp(h, clientID, "https://attacker.example.com/callback")
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid redirect url")
	})

	t.Run("a different path on the registered loopback host is refused", func(t *testing.T) {
		// Port-agnostic must not mean path-agnostic; the path is what
		// distinguishes one local client's callback from another's.
		h := newAppTestProvider(t, registered)
		rec := getApp(h, clientID, "http://127.0.0.1:53119/steal")
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "invalid redirect url")
	})

	t.Run("with no client_id the global allow-list still governs", func(t *testing.T) {
		// Non-OAuth uses of the login page must be unaffected.
		h := newAppTestProvider(t, registered)
		assert.Equal(t, http.StatusOK, getApp(h, "", "https://console.example.com/back").Code)
		assert.Equal(t, http.StatusBadRequest, getApp(h, "", "https://elsewhere.example.com/back").Code)
	})

	t.Run("an unknown client_id falls back to the allow-list", func(t *testing.T) {
		// No registry row is the legacy path the deployment's own reserved
		// client relies on.
		h := newAppTestProvider(t, registered)
		assert.Equal(t, http.StatusOK, getApp(h, "some-other-client", "https://console.example.com/back").Code)
		assert.Equal(t, http.StatusBadRequest, getApp(h, "some-other-client", loopback).Code)
	})
}

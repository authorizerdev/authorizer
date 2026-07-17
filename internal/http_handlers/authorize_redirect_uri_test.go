package http_handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	inmemorystore "github.com/authorizerdev/authorizer/internal/memory_store/in_memory"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// redirectURIClientStore is a minimal storage.Provider stub serving only
// GetClientByClientID, for exercising AuthorizeHandler's per-client
// redirect_uri exact-match enforcement.
type redirectURIClientStore struct {
	storage.Provider
	client *schemas.Client
}

func (s *redirectURIClientStore) GetClientByClientID(_ context.Context, clientID string) (*schemas.Client, error) {
	if s.client == nil || s.client.ClientID != clientID {
		return nil, errors.New("not found")
	}
	return s.client, nil
}

func newRedirectURITestProvider(t *testing.T, client *schemas.Client) *httpProvider {
	t.Helper()
	logger := zerolog.Nop()
	cfg := &config.Config{AllowedOrigins: []string{"*"}}
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

func authorizeRedirectCtx(query string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/authorize?"+query, nil)
	return c, rec
}

// TestAuthorize_RegisteredRedirectURI_RejectsUnregisteredSuffix is a
// regression test for a real security bug caught by the OIDF conformance
// suite (oidcc-ensure-registered-redirect-uri): a client with registered
// redirect URIs must reject a presented redirect_uri that isn't an exact
// match, even when it's just the registered URI plus an extra path
// segment under the same allowed host.
func TestAuthorize_RegisteredRedirectURI_RejectsUnregisteredSuffix(t *testing.T) {
	client := &schemas.Client{
		ClientID:     "client-1",
		RedirectURIs: "http://example.com/app/callback",
	}
	h := newRedirectURITestProvider(t, client)

	c, rec := authorizeRedirectCtx("client_id=client-1&state=xyz&response_type=code&response_mode=query&scope=openid&redirect_uri=" +
		"http%3A%2F%2Fexample.com%2Fapp%2Fcallback%2Funregistered-suffix")
	h.AuthorizeHandler()(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code, "an unregistered path suffix must be rejected even under an allowed host")
	assert.Contains(t, rec.Body.String(), "invalid_request")
}

func TestAuthorize_RegisteredRedirectURI_AcceptsExactMatch(t *testing.T) {
	client := &schemas.Client{
		ClientID:     "client-1",
		RedirectURIs: "http://example.com/app/callback",
	}
	h := newRedirectURITestProvider(t, client)

	c, rec := authorizeRedirectCtx("client_id=client-1&state=xyz&response_type=code&response_mode=query&scope=openid&redirect_uri=" +
		"http%3A%2F%2Fexample.com%2Fapp%2Fcallback")
	h.AuthorizeHandler()(c)

	assert.NotEqual(t, http.StatusBadRequest, rec.Code, "an exact-matching registered redirect_uri must be accepted")
}

// TestAuthorize_NoRegisteredRedirectURIs_FallsBackToOriginCheck preserves
// today's behavior for clients that have never registered redirect_uris
// (the only kind that exist today, since no API surface sets the field) —
// this fix must not break them.
func TestAuthorize_NoRegisteredRedirectURIs_FallsBackToOriginCheck(t *testing.T) {
	client := &schemas.Client{ClientID: "client-1"} // RedirectURIs empty
	h := newRedirectURITestProvider(t, client)

	c, rec := authorizeRedirectCtx("client_id=client-1&state=xyz&response_type=code&response_mode=query&scope=openid&redirect_uri=" +
		"http%3A%2F%2Fexample.com%2Fapp%2Fanything")
	h.AuthorizeHandler()(c)

	assert.NotEqual(t, http.StatusBadRequest, rec.Code, "clients with no registered redirect_uris keep the existing origin-only check")
}

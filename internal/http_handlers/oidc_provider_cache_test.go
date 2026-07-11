package http_handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newDiscoveryServer serves a minimal OIDC discovery document whose issuer
// matches its own URL, and counts how many times the discovery doc is fetched.
func newDiscoveryServer(t *testing.T) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 srv.URL,
			"authorization_endpoint": srv.URL + "/authorize",
			"token_endpoint":         srv.URL + "/token",
			"jwks_uri":               srv.URL + "/jwks",
		})
	})
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestGetOIDCProviderCachesByIssuer(t *testing.T) {
	ctx := context.Background()

	srv1, hits1 := newDiscoveryServer(t)
	srv2, _ := newDiscoveryServer(t)

	// Same issuer -> same cached *oidc.Provider instance, discovery fetched once.
	p1a, err := getOIDCProvider(ctx, srv1.URL)
	require.NoError(t, err)
	p1b, err := getOIDCProvider(ctx, srv1.URL)
	require.NoError(t, err)
	assert.Same(t, p1a, p1b, "repeat calls for the same issuer must return the same instance")
	assert.Equal(t, int32(1), atomic.LoadInt32(hits1), "discovery doc must be fetched only once per issuer")

	// Different issuer -> different instance.
	p2, err := getOIDCProvider(ctx, srv2.URL)
	require.NoError(t, err)
	assert.NotSame(t, p1a, p2, "a different issuer must return a different instance")
}

package clientauth

// Tests for loadJWKS's caching contract (client_assertion.go): the H7
// guarantee that two TrustedIssuer rows never share a cached JWKS even when
// they point at the same URL, that a cache hit avoids a second fetch, and
// that an expired/evicted entry triggers a refetch. The rest of this
// package's tests cover assertion-verification logic but never assert
// anything about the cache itself.

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// countingFetchURL wraps a fixed JWKS payload in a fetchURL stub that counts
// every invocation, so a test can distinguish a cache hit from a real fetch.
func countingFetchURL(jwks []byte) (fn func(ctx context.Context, rawURL string) ([]byte, error), calls *int) {
	n := 0
	return func(_ context.Context, _ string) ([]byte, error) {
		n++
		return jwks, nil
	}, &n
}

// jwksProvider builds a bare clientauth provider wired only with the two
// fields loadJWKS touches: MemoryStoreProvider and fetchURL. No
// StorageProvider is needed — loadJWKS never calls it.
func jwksProvider(t *testing.T, fetchURL func(ctx context.Context, rawURL string) ([]byte, error)) *provider {
	t.Helper()
	logger := zerolog.Nop()
	p := New(
		&config.Config{ClientID: "reserved", ClientSecret: "reserved-secret"},
		&Dependencies{Log: &logger, MemoryStoreProvider: newFakeMemStore()},
	).(*provider)
	p.fetchURL = fetchURL
	return p
}

func staticIssuer(id, jwksURL string) *schemas.TrustedIssuer {
	return &schemas.TrustedIssuer{
		ID:            id,
		KeySourceType: constants.KeySourceStaticJWKSURL,
		JWKSUrl:       refString(jwksURL),
	}
}

func TestJWKSCache_PerIssuerIsolation(t *testing.T) {
	key := genKey(t)
	jwks := jwksBytes(t, &key.PublicKey, testKID)
	fetchURL, calls := countingFetchURL(jwks)
	p := jwksProvider(t, fetchURL)

	const sharedURL = "https://issuer.test.example.com/jwks.json"
	issuerA := staticIssuer("issuer-a", sharedURL)
	issuerB := staticIssuer("issuer-b", sharedURL)

	_, err := p.loadJWKS(context.Background(), issuerA)
	require.NoError(t, err)
	_, err = p.loadJWKS(context.Background(), issuerB)
	require.NoError(t, err)

	// A cache key derived from the URL alone would let issuerB's lookup hit
	// issuerA's cache entry (an H7 violation). Two rows sharing a JWKS URL must
	// each fetch on their own first load.
	assert.Equal(t, 2, *calls, "two distinct TrustedIssuer rows sharing a JWKS URL must each trigger their own fetch")

	store := p.MemoryStoreProvider.(*fakeMemStore)
	store.mu.Lock()
	_, hasA := store.cache["jwks_cache:issuer-a"]
	_, hasB := store.cache["jwks_cache:issuer-b"]
	store.mu.Unlock()
	assert.True(t, hasA, "issuer-a must have its own cache entry")
	assert.True(t, hasB, "issuer-b must have its own cache entry")
}

func TestJWKSCache_HitAvoidsRefetch(t *testing.T) {
	key := genKey(t)
	jwks := jwksBytes(t, &key.PublicKey, testKID)
	fetchURL, calls := countingFetchURL(jwks)
	p := jwksProvider(t, fetchURL)

	issuer := staticIssuer("issuer-a", "https://issuer.test.example.com/jwks.json")

	_, err := p.loadJWKS(context.Background(), issuer)
	require.NoError(t, err)
	_, err = p.loadJWKS(context.Background(), issuer)
	require.NoError(t, err)

	assert.Equal(t, 1, *calls, "a second load within the TTL must be served from cache, not refetched")
}

func TestJWKSCache_RefetchesAfterEviction(t *testing.T) {
	key := genKey(t)
	jwks := jwksBytes(t, &key.PublicKey, testKID)
	fetchURL, calls := countingFetchURL(jwks)
	p := jwksProvider(t, fetchURL)

	issuer := staticIssuer("issuer-a", "https://issuer.test.example.com/jwks.json")

	_, err := p.loadJWKS(context.Background(), issuer)
	require.NoError(t, err)
	require.Equal(t, 1, *calls)

	// jwksCacheTTLSeconds is a hardcoded package const with no test seam to
	// shorten it (finding, not a bug — see task report). Expiry is simulated by
	// evicting the entry directly from the fake store, exactly what a real TTL
	// store does once the deadline passes.
	store := p.MemoryStoreProvider.(*fakeMemStore)
	store.mu.Lock()
	delete(store.cache, "jwks_cache:issuer-a")
	store.mu.Unlock()

	_, err = p.loadJWKS(context.Background(), issuer)
	require.NoError(t, err)
	assert.Equal(t, 2, *calls, "loadJWKS must refetch once the cache entry has expired/evicted")
}

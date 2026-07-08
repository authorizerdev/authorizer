package clientauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// plainClientSeam replaces the SSRF-hardened client factory with a plain client
// so the real performTokenReview HTTP logic can reach an httptest apiserver
// (validators.SafeHTTPClient refuses loopback, so httptest is otherwise
// unreachable — the same reason JWKS tests stub fetchURL).
func plainClientSeam(_ context.Context, _ string, _ time.Duration) (*http.Client, error) {
	return &http.Client{Timeout: 2 * time.Second}, nil
}

// tokenReviewServer returns an httptest server that answers the TokenReview
// subresource with status.authenticated = authenticated.
func tokenReviewServer(t *testing.T, authenticated bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, tokenReviewPath, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if authenticated {
			_, _ = w.Write([]byte(`{"apiVersion":"authentication.k8s.io/v1","kind":"TokenReview","status":{"authenticated":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":{"authenticated":false,"error":"pod not found"}}`))
	}))
}

// --- performTokenReview unit tests (real HTTP round-trip against httptest) ---

func TestPerformTokenReview_AuthenticatedTruePasses(t *testing.T) {
	srv := tokenReviewServer(t, true)
	defer srv.Close()
	p := &provider{safeHTTPClient: plainClientSeam}
	err := p.performTokenReview(context.Background(), srv.URL, "the-projected-token", testExpectedAud)
	assert.NoError(t, err)
}

func TestPerformTokenReview_AuthenticatedFalseFailsClosed(t *testing.T) {
	srv := tokenReviewServer(t, false)
	defer srv.Close()
	p := &provider{safeHTTPClient: plainClientSeam}
	err := p.performTokenReview(context.Background(), srv.URL, "the-projected-token", testExpectedAud)
	require.Error(t, err, "authenticated=false must fail closed")
}

func TestPerformTokenReview_Non2xxFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	p := &provider{safeHTTPClient: plainClientSeam}
	err := p.performTokenReview(context.Background(), srv.URL, "tok", testExpectedAud)
	require.Error(t, err, "a non-2xx apiserver response must fail closed")
}

// --- ResolveClient integration: EnableTokenReview wiring ---

func TestClientAssertion_TokenReviewAuthenticatedTruePasses(t *testing.T) {
	srv := tokenReviewServer(t, true)
	defer srv.Close()

	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	r.safeHTTPClient = plainClientSeam
	iss := r.StorageProvider.(*assertionStore).issuers[testIssuerURL]
	iss.EnableTokenReview = true
	iss.KubernetesAPIServerURL = refString(srv.URL)

	client, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, validClaims())))
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, testSAClientPK, client.ID)
}

func TestClientAssertion_TokenReviewAuthenticatedFalseRejected(t *testing.T) {
	srv := tokenReviewServer(t, false)
	defer srv.Close()

	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	r.safeHTTPClient = plainClientSeam
	iss := r.StorageProvider.(*assertionStore).issuers[testIssuerURL]
	iss.EnableTokenReview = true
	iss.KubernetesAPIServerURL = refString(srv.URL)

	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, validClaims())))
	assert.ErrorIs(t, err, ErrInvalidClient, "a deleted-object token (authenticated=false) must be rejected")
}

// TestClientAssertion_TokenReviewMissingAPIServerURLRejected: EnableTokenReview
// with no apiserver URL fails closed without any network call.
func TestClientAssertion_TokenReviewMissingAPIServerURLRejected(t *testing.T) {
	key := genKey(t)
	r := buildResolver(t, jwksBytes(t, &key.PublicKey, testKID), testSubject)
	r.safeHTTPClient = func(context.Context, string, time.Duration) (*http.Client, error) {
		t.Fatal("safeHTTPClient must not be called when kubernetes_api_server_url is empty")
		return nil, nil
	}
	r.StorageProvider.(*assertionStore).issuers[testIssuerURL].EnableTokenReview = true

	_, err := r.ResolveClient(context.Background(), assertionParams(signRS256(t, key, testKID, validClaims())))
	assert.ErrorIs(t, err, ErrInvalidClient)
}

package validators

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSafeHTTPClient_RejectsPrivateIP is the pre-existing-behavior regression
// guard: with no bypass involved, a private/loopback target is still rejected.
func TestSafeHTTPClient_RejectsPrivateIP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := SafeHTTPClient(context.Background(), srv.URL, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private")
}

// TestSafeHTTPClientAllowPrivate_AllowsPrivateIP proves the SSO-broker-only
// variant actually reaches a loopback target end-to-end (not just "no error
// from the constructor" — the constructor doesn't dial anything itself).
func TestSafeHTTPClientAllowPrivate_AllowsPrivateIP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client, err := SafeHTTPClientAllowPrivate(context.Background(), srv.URL, time.Second)
	require.NoError(t, err)

	resp, err := client.Get(srv.URL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(body))
}

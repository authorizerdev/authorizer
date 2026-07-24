package events

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

// Webhook custom headers are stored as a JSON object (the GraphQL Map scalar),
// so an admin can persist a value that is a number, bool, null or nested object
// (e.g. {"X-Retry": 3}). RegisterEvent's header loop previously did
// val.(string) — a single-return assertion that PANICS on any non-string value.
// That loop runs inside a bare `go func()` (see the RegisterEvent call sites),
// so the panic is unrecovered and crashes the whole process.
//
// headerValueString coerces the value instead. These cases cover the exact
// inputs that used to panic.
func TestHeaderValueString_NonStringDoesNotPanic(t *testing.T) {
	require.NotPanics(t, func() {
		require.Equal(t, "hello", headerValueString("hello"))
		require.Equal(t, "", headerValueString(nil))
		require.Equal(t, "3", headerValueString(float64(3))) // JSON numbers decode to float64
		require.Equal(t, "true", headerValueString(true))
	})
}

// TestWebhookHTTPClient_AllowPrivateFalse_RejectsPrivateIP is the production-default
// no-op guard: with allowPrivate=false (Config.TestAllowPrivateWebhookHosts unset,
// the production default) a private/loopback webhook endpoint is still rejected
// exactly as before the escape hatch existed. deliver() takes this same false path
// whenever the flag is absent, so production delivery is unchanged.
func TestWebhookHTTPClient_AllowPrivateFalse_RejectsPrivateIP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := webhookHTTPClient(context.Background(), srv.URL, time.Second, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "private")
}

// TestWebhookHTTPClient_AllowPrivateTrue_ReachesLoopback proves the escape hatch
// actually reaches a loopback target end-to-end when explicitly opted into.
func TestWebhookHTTPClient_AllowPrivateTrue_ReachesLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client, err := webhookHTTPClient(context.Background(), srv.URL, time.Second, true)
	require.NoError(t, err)

	resp, err := client.Get(srv.URL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(body))
}

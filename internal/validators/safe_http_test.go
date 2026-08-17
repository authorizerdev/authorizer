package validators

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestSafeHTTPClient_RejectsEverySpecialPurposeRange walks the IANA
// special-purpose ranges as a table, so the block list is asserted as a SET
// rather than through whichever range a given test happened to pick.
//
// It exists because two of these were missing and nothing noticed: all three
// TEST-NETs were blocked while 198.18.0.0/15 and 192.88.99.0/24 — from the same
// registry, equally non-routable — were reachable. A per-range table makes the
// next omission visible as a missing row instead of an absence of evidence.
//
// 198.18.0.0/15 is the one with real exposure. Not being globally routable is
// exactly why organisations use it internally — as a lab range, or to number a
// network they do not want colliding with RFC 1918 — so an SSRF there reaches a
// live internal service via an address that reads as public.
func TestSafeHTTPClient_RejectsEverySpecialPurposeRange(t *testing.T) {
	cases := []struct{ ip, why string }{
		{"10.1.2.3", "RFC 1918"},
		{"172.16.5.5", "RFC 1918"},
		{"192.168.1.1", "RFC 1918"},
		{"127.0.0.1", "loopback"},
		{"169.254.169.254", "link-local / cloud metadata"},
		{"100.64.1.1", "CGNAT"},
		{"0.0.0.5", "this network"},
		{"192.0.0.5", "IETF protocol assignments"},
		{"192.0.2.5", "TEST-NET-1"},
		{"198.51.100.5", "TEST-NET-2"},
		{"203.0.113.5", "TEST-NET-3"},
		{"198.18.0.5", "RFC 2544 benchmarking"},
		{"198.19.255.5", "RFC 2544 benchmarking, upper half of the /15"},
		{"192.88.99.5", "6to4 relay anycast (RFC 7526)"},
		{"224.0.0.5", "multicast"},
		{"240.0.0.5", "reserved"},
		{"::1", "IPv6 loopback"},
		{"fc00::5", "IPv6 ULA"},
		{"fe80::5", "IPv6 link-local"},
	}
	for _, tc := range cases {
		t.Run(tc.ip, func(t *testing.T) {
			_, err := SafeHTTPClient(context.Background(), "https://"+hostFor(tc.ip)+"/x", time.Second)
			require.Error(t, err, "%s (%s) must be refused", tc.ip, tc.why)
			assert.Contains(t, err.Error(), "private/internal networks are not allowed",
				"%s (%s) must be refused by the range check, not by a DNS or TLS failure "+
					"that would mask a missing range", tc.ip, tc.why)
		})
	}
}

// hostFor brackets IPv6 literals for use in a URL authority.
func hostFor(ip string) string {
	if strings.Contains(ip, ":") {
		return "[" + ip + "]"
	}
	return ip
}

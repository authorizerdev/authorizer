package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// req builds a request with the given peer address and headers.
func req(remoteAddr string, headers map[string]string) *http.Request {
	r := &http.Request{Header: http.Header{}, RemoteAddr: remoteAddr}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

// resetTrustedProxies restores the trust-nothing default so cases don't leak
// into each other.
func resetTrustedProxies(t *testing.T) {
	t.Helper()
	require.NoError(t, SetTrustedProxies(nil))
}

// TestGetIPIgnoresForwardedHeadersFromUntrustedPeer is the regression test for
// GHSA-93hc-xq3w-xw87. The admin-secret lockout buckets on GetIP, so a caller
// that can choose its own bucket key gets an unlimited guessing budget.
func TestGetIPIgnoresForwardedHeadersFromUntrustedPeer(t *testing.T) {
	resetTrustedProxies(t)

	t.Run("X-Forwarded-For is ignored", func(t *testing.T) {
		got := GetIP(req("1.2.3.4:51001", map[string]string{"X-Forwarded-For": "203.0.113.9"}))
		assert.Equal(t, "1.2.3.4", got)
	})

	t.Run("X-Real-Ip is ignored", func(t *testing.T) {
		got := GetIP(req("1.2.3.4:51001", map[string]string{"X-Real-Ip": "203.0.113.9"}))
		assert.Equal(t, "1.2.3.4", got)
	})

	t.Run("rotating the header does not rotate the bucket key", func(t *testing.T) {
		// The exploit shape: same attacker, a different forged header per
		// request. Every call must return the SAME key or the lockout never
		// fills.
		first := GetIP(req("1.2.3.4:51001", map[string]string{"X-Forwarded-For": "203.0.113.1"}))
		for _, forged := range []string{"203.0.113.2", "198.51.100.7", "10.0.0.1"} {
			got := GetIP(req("1.2.3.4:51001", map[string]string{"X-Forwarded-For": forged}))
			assert.Equal(t, first, got, "forged %q produced a different lockout bucket", forged)
		}
	})

	t.Run("a new connection does not rotate the bucket key", func(t *testing.T) {
		// Second, independent bypass: RemoteAddr is "IP:port", so returning it
		// verbatim gave every TCP connection its own bucket even with no
		// forwarded headers at all.
		a := GetIP(req("1.2.3.4:51001", nil))
		b := GetIP(req("1.2.3.4:51002", nil))
		assert.Equal(t, a, b)
		assert.Equal(t, "1.2.3.4", a)
	})
}

func TestGetIPHonoursForwardedHeadersFromTrustedProxy(t *testing.T) {
	t.Cleanup(func() { resetTrustedProxies(t) })

	t.Run("single trusted proxy yields the real client", func(t *testing.T) {
		require.NoError(t, SetTrustedProxies([]string{"10.0.0.0/8"}))
		got := GetIP(req("10.0.0.5:443", map[string]string{"X-Forwarded-For": "1.1.1.1"}))
		assert.Equal(t, "1.1.1.1", got)
	})

	t.Run("bare IP entry is treated as a host route", func(t *testing.T) {
		require.NoError(t, SetTrustedProxies([]string{"10.0.0.5"}))
		assert.Equal(t, "1.1.1.1", GetIP(req("10.0.0.5:443", map[string]string{"X-Forwarded-For": "1.1.1.1"})))
		assert.Equal(t, "10.0.0.6", GetIP(req("10.0.0.6:443", map[string]string{"X-Forwarded-For": "1.1.1.1"})))
	})

	t.Run("CDN in front of LB: both hops trusted yields the client", func(t *testing.T) {
		// client 1.1.1.1 -> CDN 203.0.113.7 -> LB 10.0.0.5 -> authorizer
		require.NoError(t, SetTrustedProxies([]string{"10.0.0.0/8", "203.0.113.7/32"}))
		got := GetIP(req("10.0.0.5:443", map[string]string{"X-Forwarded-For": "1.1.1.1, 203.0.113.7"}))
		assert.Equal(t, "1.1.1.1", got)
	})

	t.Run("CDN in front of LB: only the LB trusted stops at the CDN", func(t *testing.T) {
		// Documents the operator-visible consequence: an unlisted hop is where
		// the walk stops. Returning the CDN address is correct — nothing
		// vouched for the entry to its left.
		require.NoError(t, SetTrustedProxies([]string{"10.0.0.0/8"}))
		got := GetIP(req("10.0.0.5:443", map[string]string{"X-Forwarded-For": "1.1.1.1, 203.0.113.7"}))
		assert.Equal(t, "203.0.113.7", got)
	})

	t.Run("X-Forwarded-For wins over X-Real-Ip", func(t *testing.T) {
		require.NoError(t, SetTrustedProxies([]string{"10.0.0.0/8"}))
		got := GetIP(req("10.0.0.5:443", map[string]string{
			"X-Forwarded-For": "1.1.1.1",
			"X-Real-Ip":       "2.2.2.2",
		}))
		assert.Equal(t, "1.1.1.1", got)
	})

	t.Run("X-Real-Ip is used when X-Forwarded-For is absent", func(t *testing.T) {
		require.NoError(t, SetTrustedProxies([]string{"10.0.0.0/8"}))
		got := GetIP(req("10.0.0.5:443", map[string]string{"X-Real-Ip": "2.2.2.2"}))
		assert.Equal(t, "2.2.2.2", got)
	})

	t.Run("malformed entry falls back to the peer", func(t *testing.T) {
		require.NoError(t, SetTrustedProxies([]string{"10.0.0.0/8"}))
		got := GetIP(req("10.0.0.5:443", map[string]string{"X-Forwarded-For": "not-an-ip"}))
		assert.Equal(t, "10.0.0.5", got)
	})

	t.Run("IPv6 peer", func(t *testing.T) {
		require.NoError(t, SetTrustedProxies([]string{"2001:db8::/32"}))
		got := GetIP(req("[2001:db8::1]:443", map[string]string{"X-Forwarded-For": "1.1.1.1"}))
		assert.Equal(t, "1.1.1.1", got)
	})
}

func TestSetTrustedProxiesRejectsInvalidEntry(t *testing.T) {
	t.Cleanup(func() { resetTrustedProxies(t) })
	require.NoError(t, SetTrustedProxies([]string{"10.0.0.0/8"}))

	// A typo must not silently widen or clear trust.
	require.Error(t, SetTrustedProxies([]string{"10.0.0.0/8", "not-a-cidr"}))
	assert.Equal(t, "1.1.1.1", GetIP(req("10.0.0.5:443", map[string]string{"X-Forwarded-For": "1.1.1.1"})),
		"previous configuration should survive a rejected update")
}

func TestGetIPNilRequest(t *testing.T) {
	resetTrustedProxies(t)
	assert.Equal(t, "", GetIP(nil))
}

// TestGetIPMatchesGinClientIP is the anti-drift guard. GetIP reimplements
// gin's Context.ClientIP because gin's version needs an *Engine this codebase
// does not always have (see GetIP's doc comment). This runs the same inputs
// through a real gin engine and requires identical answers, so a future change
// to either side that separates them fails here rather than in production —
// which is exactly how GHSA-93hc-xq3w-xw87 happened.
func TestGetIPMatchesGinClientIP(t *testing.T) {
	t.Cleanup(func() { resetTrustedProxies(t) })
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		trusted    []string
		remoteAddr string
		headers    map[string]string
	}{
		{"untrusted peer with xff", nil, "1.2.3.4:51001", map[string]string{"X-Forwarded-For": "203.0.113.9"}},
		{"untrusted peer with real-ip", nil, "1.2.3.4:51001", map[string]string{"X-Real-Ip": "203.0.113.9"}},
		{"untrusted peer no headers", nil, "1.2.3.4:51001", nil},
		{"trusted single hop", []string{"10.0.0.0/8"}, "10.0.0.5:443", map[string]string{"X-Forwarded-For": "1.1.1.1"}},
		{"trusted two hops both listed", []string{"10.0.0.0/8", "203.0.113.7/32"}, "10.0.0.5:443", map[string]string{"X-Forwarded-For": "1.1.1.1, 203.0.113.7"}},
		{"trusted two hops one listed", []string{"10.0.0.0/8"}, "10.0.0.5:443", map[string]string{"X-Forwarded-For": "1.1.1.1, 203.0.113.7"}},
		{"trusted xff beats real-ip", []string{"10.0.0.0/8"}, "10.0.0.5:443", map[string]string{"X-Forwarded-For": "1.1.1.1", "X-Real-Ip": "2.2.2.2"}},
		{"trusted real-ip only", []string{"10.0.0.0/8"}, "10.0.0.5:443", map[string]string{"X-Real-Ip": "2.2.2.2"}},
		{"trusted malformed xff", []string{"10.0.0.0/8"}, "10.0.0.5:443", map[string]string{"X-Forwarded-For": "not-an-ip"}},
		{"bare ip entry", []string{"10.0.0.5"}, "10.0.0.5:443", map[string]string{"X-Forwarded-For": "1.1.1.1"}},
		{"ipv6 trusted peer", []string{"2001:db8::/32"}, "[2001:db8::1]:443", map[string]string{"X-Forwarded-For": "1.1.1.1"}},
		{"leftmost entry is trusted", []string{"10.0.0.0/8"}, "10.0.0.5:443", map[string]string{"X-Forwarded-For": "10.0.0.9"}},
		{"trusted malformed real-ip", []string{"10.0.0.0/8"}, "10.0.0.5:443", map[string]string{"X-Real-Ip": "not-an-ip"}},
		{"trusted malformed real-ip and xff", []string{"10.0.0.0/8"}, "10.0.0.5:443", map[string]string{"X-Forwarded-For": "bogus", "X-Real-Ip": "also-bogus"}},
		{"trusted empty xff", []string{"10.0.0.0/8"}, "10.0.0.5:443", map[string]string{"X-Forwarded-For": ""}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, SetTrustedProxies(tc.trusted))

			engine := gin.New()
			require.NoError(t, engine.SetTrustedProxies(tc.trusted))

			r := req(tc.remoteAddr, tc.headers)
			var ginAnswer string
			engine.GET("/", func(c *gin.Context) { ginAnswer = c.ClientIP() })
			w := httptest.NewRecorder()
			probe := httptest.NewRequest(http.MethodGet, "/", nil)
			probe.RemoteAddr = tc.remoteAddr
			for k, v := range tc.headers {
				probe.Header.Set(k, v)
			}
			engine.ServeHTTP(w, probe)

			assert.Equal(t, ginAnswer, GetIP(r),
				"utils.GetIP and gin Context.ClientIP disagree — the router and the lockout would bucket differently")
		})
	}
}

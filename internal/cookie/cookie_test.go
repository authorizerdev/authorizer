package cookie

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// mfaCookieTestExpiry is a fixed, comfortably-in-the-future expiry the MFA
// cookie tests use to compute an expected MaxAge without a flaky exact-second
// comparison against time.Now() inside the assertion.
func mfaCookieTestExpiry(d time.Duration) int64 {
	return time.Now().Add(d).Unix()
}

func TestBuildSessionCookies(t *testing.T) {
	tests := []struct {
		name       string
		hostname   string
		secure     bool
		sameSite   http.SameSite
		wantDomain string // expected `.example.com`-style domain on the domain-scoped cookie
	}{
		{"production https", "https://auth.example.com", true, http.SameSiteNoneMode, ".example.com"},
		{"localhost dev", "http://localhost:8080", false, http.SameSiteLaxMode, "localhost"},
		{"subdomain", "https://auth.svc.example.com", true, http.SameSiteStrictMode, ".example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cookies := BuildSessionCookies(tt.hostname, "session-id", tt.secure, tt.sameSite)
			require.Len(t, cookies, 2, "BuildSessionCookies must return exactly the host-scoped and domain-scoped pair")

			for _, c := range cookies {
				assert.Equal(t, "session-id", c.Value)
				assert.Equal(t, tt.secure, c.Secure)
				assert.True(t, c.HttpOnly, "session cookies must be HttpOnly")
				assert.Equal(t, "/", c.Path)
				assert.Equal(t, tt.sameSite, c.SameSite)
				assert.Equal(t, 24*60*60, c.MaxAge, "session cookie MaxAge must be 1 day")
			}

			// Sanity-check cookie names.
			assert.Equal(t, constants.AppCookieName+"_session", cookies[0].Name)
			assert.Equal(t, constants.AppCookieName+"_session_domain", cookies[1].Name)
			// Domain-scoped cookie picks up the apex.
			assert.Equal(t, tt.wantDomain, cookies[1].Domain)
		})
	}
}

func TestBuildMfaSessionCookies(t *testing.T) {
	expiresAt := mfaCookieTestExpiry(3 * time.Minute)
	cookies := BuildMfaSessionCookies("https://auth.example.com", "mfa-id", true, expiresAt)
	require.Len(t, cookies, 2)
	for _, c := range cookies {
		assert.Equal(t, "mfa-id", c.Value)
		assert.True(t, c.Secure)
		assert.True(t, c.HttpOnly)
		assert.Equal(t, http.SameSiteNoneMode, c.SameSite, "secure → SameSite=None")
		assert.InDelta(t, 180, c.MaxAge, 2, "MaxAge must track the caller's actual session expiry, not a hardcoded value")
	}
	assert.Equal(t, constants.MfaCookieName+"_session", cookies[0].Name)
	assert.Equal(t, constants.MfaCookieName+"_session_domain", cookies[1].Name)
}

// TestBuildMfaSessionCookies_MaxAgeTracksExpiry guards the fix for a real bug
// this caught: the cookie's MaxAge used to be hardcoded to 60s regardless of
// the expiresAt passed to MemoryStoreProvider.SetMfaSession (1-3 minutes
// depending on caller). A user who took longer than 60s to act on an MFA
// offer/verify screen would get "invalid session" even though the underlying
// session was still valid - the cookie carrying it to the browser had already
// been deleted. MaxAge must vary with expiresAt, not stay constant.
func TestBuildMfaSessionCookies_MaxAgeTracksExpiry(t *testing.T) {
	oneMinute := BuildMfaSessionCookies("https://auth.example.com", "mfa-id", true, mfaCookieTestExpiry(1*time.Minute))
	threeMinutes := BuildMfaSessionCookies("https://auth.example.com", "mfa-id", true, mfaCookieTestExpiry(3*time.Minute))
	assert.InDelta(t, 60, oneMinute[0].MaxAge, 2)
	assert.InDelta(t, 180, threeMinutes[0].MaxAge, 2)
	assert.Greater(t, threeMinutes[0].MaxAge, oneMinute[0].MaxAge, "a longer session expiry must produce a longer-lived cookie")
}

func TestBuildMfaSessionCookies_InsecureLaxSameSite(t *testing.T) {
	cookies := BuildMfaSessionCookies("http://localhost:8080", "mfa-id", false, mfaCookieTestExpiry(3*time.Minute))
	require.Len(t, cookies, 2)
	for _, c := range cookies {
		assert.False(t, c.Secure)
		// Insecure → SameSite=Lax (so cross-site flows still complete when not behind TLS).
		// Verified against the original SetMfaSession behaviour: this is intentional.
		assert.Equal(t, http.SameSiteLaxMode, c.SameSite)
	}
}

func TestParseSameSite(t *testing.T) {
	tests := []struct {
		in   string
		want http.SameSite
	}{
		{"none", http.SameSiteNoneMode},
		{"NONE", http.SameSiteNoneMode},
		{"strict", http.SameSiteStrictMode},
		{"lax", http.SameSiteLaxMode},
		{"", http.SameSiteLaxMode}, // unknown defaults to Lax
		{"garbage", http.SameSiteLaxMode},
		{"  none  ", http.SameSiteNoneMode},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseSameSite(tt.in))
		})
	}
}

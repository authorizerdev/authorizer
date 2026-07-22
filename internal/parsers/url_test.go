package parsers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeAuthorizerURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid https", "https://auth.example.com", "https://auth.example.com"},
		{"valid http", "http://localhost:8080", "http://localhost:8080"},
		{"strips path", "https://auth.example.com/callback", "https://auth.example.com"},
		{"strips query", "https://auth.example.com?evil=1", "https://auth.example.com"},
		{"strips fragment", "https://auth.example.com#frag", "https://auth.example.com"},
		{"strips trailing slash", "https://auth.example.com/", "https://auth.example.com"},
		{"rejects javascript scheme", "javascript:alert(1)", ""},
		{"rejects ftp scheme", "ftp://evil.com", ""},
		{"rejects no scheme", "evil.com", ""},
		{"rejects empty", "", ""},
		{"rejects userinfo", "https://user:pass@evil.com", ""},
		{"valid with port", "https://auth.example.com:443", "https://auth.example.com:443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeAuthorizerURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeHost(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid host", "auth.example.com", "auth.example.com"},
		{"valid host with port", "localhost:8080", "localhost:8080"},
		{"rejects path", "evil.com/path", ""},
		{"rejects query", "evil.com?q=1", ""},
		{"rejects fragment", "evil.com#f", ""},
		{"rejects at sign", "user@evil.com", ""},
		{"rejects backslash", "evil.com\\path", ""},
		{"rejects newline", "evil.com\nX-Injected: true", ""},
		{"rejects carriage return", "evil.com\rX-Injected: true", ""},
		{"empty string", "", ""},
		{"whitespace only", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeHost(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetHostFromRequest(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		host    string
		want    string
	}{
		{
			name: "X-Authorizer-URL takes priority",
			headers: map[string]string{
				"X-Authorizer-URL":  "https://auth.example.com",
				"X-Forwarded-Proto": "http",
				"X-Forwarded-Host":  "ignored.example.com",
			},
			host: "request.example.com",
			want: "https://auth.example.com",
		},
		{
			name:    "falls back to X-Forwarded-Proto + X-Forwarded-Host",
			headers: map[string]string{"X-Forwarded-Proto": "https", "X-Forwarded-Host": "edge.example.com"},
			host:    "internal.example.com",
			want:    "https://edge.example.com",
		},
		{
			name: "ignores invalid X-Authorizer-URL",
			headers: map[string]string{
				"X-Authorizer-URL":  "user:pass@evil.example.com",
				"X-Forwarded-Proto": "https",
				"X-Forwarded-Host":  "edge.example.com",
			},
			host: "ignored",
			want: "https://edge.example.com",
		},
		{
			name:    "falls back to Request.Host",
			headers: map[string]string{},
			host:    "auth.example.com",
			want:    "http://auth.example.com",
		},
		{
			name:    "defaults to localhost when nothing is set",
			headers: map[string]string{},
			host:    "",
			want:    "http://localhost",
		},
		{
			name:    "rejects spoofed X-Forwarded-Host with path injection",
			headers: map[string]string{"X-Forwarded-Host": "evil.example.com/path"},
			host:    "auth.example.com",
			want:    "http://auth.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Host: tt.host, Header: http.Header{}}
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			assert.Equal(t, tt.want, GetHostFromRequest(r))
		})
	}
}

func TestGetAppURLFromRequest(t *testing.T) {
	r := &http.Request{Host: "auth.example.com", Header: http.Header{}}
	assert.Equal(t, "http://auth.example.com/app", GetAppURLFromRequest(r))
}

// TestGetHostFromRequestTrustedURL is the regression guard for the
// host-header-injection account-takeover fix (CWE-640): when --url is
// configured, no request header may control the server's own base URL, and
// when it is NOT configured the legacy header-based behavior is unchanged.
func TestGetHostFromRequestTrustedURL(t *testing.T) {
	// Every attacker-controllable host source points at evil.com.
	spoofed := func() *http.Request {
		r := &http.Request{Host: "evil.example.com", Header: http.Header{}}
		r.Header.Set("X-Authorizer-URL", "https://evil.example.com")
		r.Header.Set("X-Forwarded-Proto", "https")
		r.Header.Set("X-Forwarded-Host", "evil.example.com")
		return r
	}

	t.Run("trusted URL overrides every spoofed header", func(t *testing.T) {
		SetTrustedURL("https://auth.example.com")
		defer SetTrustedURL("")

		assert.Equal(t, "https://auth.example.com", GetHostFromRequest(spoofed()),
			"a configured trusted URL must win over X-Authorizer-URL / X-Forwarded-Host / Host")
		// The email-link builder rides on the same helper, so the reset/verify/
		// magic-link URL host is now the trusted host, not the attacker's.
		assert.Equal(t, "https://auth.example.com/app", GetAppURLFromRequest(spoofed()))
	})

	t.Run("trusted URL is normalized to scheme+host", func(t *testing.T) {
		SetTrustedURL("https://auth.example.com/some/path/")
		defer SetTrustedURL("")
		assert.Equal(t, "https://auth.example.com", GetHostFromRequest(spoofed()))
	})

	t.Run("invalid trusted URL is treated as unset (falls back to headers)", func(t *testing.T) {
		SetTrustedURL("not-a-url")
		defer SetTrustedURL("")
		// sanitizeAuthorizerURL rejects it, so header-based derivation resumes.
		assert.Equal(t, "https://evil.example.com", GetHostFromRequest(spoofed()))
	})

	t.Run("no regression: header behavior unchanged when trusted URL is unset", func(t *testing.T) {
		// Explicitly ensure the global is clear (defaults to empty, but be
		// robust to test ordering).
		SetTrustedURL("")
		assert.Equal(t, "https://evil.example.com", GetHostFromRequest(spoofed()),
			"with no --url configured, the legacy X-Authorizer-URL priority must be preserved")
	})
}

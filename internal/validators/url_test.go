package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeOrigin(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"https with port 443", "https://example.com:443", "example.com"},
		{"http with port 80", "http://example.com:80", "example.com"},
		{"https without port", "https://example.com", "example.com"},
		{"http without port", "http://example.com", "example.com"},
		{"with custom port", "http://localhost:3000", "localhost:3000"},
		{"bare domain", "example.com", "example.com"},
		{"bare domain with port", "localhost:8080", "localhost:8080"},
		{"with path", "https://example.com/callback", "example.com"},
		{"with path and port", "http://localhost:3000/app/login", "localhost:3000"},
		{"subdomain", "https://auth.example.com", "auth.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeOrigin(tt.input))
		})
	}
}

func TestIsValidOrigin(t *testing.T) {
	t.Run("wildcard allows everything", func(t *testing.T) {
		assert.True(t, IsValidOrigin("https://anything.com", []string{"*"}))
		assert.True(t, IsValidOrigin("http://localhost:9999", []string{"*"}))
	})

	t.Run("empty config defaults to wildcard", func(t *testing.T) {
		assert.True(t, IsValidOrigin("https://anything.com", []string{}))
		assert.True(t, IsValidOrigin("https://anything.com", nil))
	})

	// --- Exact domain matching ---

	t.Run("exact domain without port", func(t *testing.T) {
		allowed := []string{"https://example.com"}
		assert.True(t, IsValidOrigin("https://example.com", allowed))
		assert.True(t, IsValidOrigin("https://example.com/callback", allowed))
		assert.True(t, IsValidOrigin("http://example.com", allowed))
		assert.False(t, IsValidOrigin("https://evil.com", allowed))
		assert.False(t, IsValidOrigin("https://sub.example.com", allowed))
	})

	t.Run("exact domain with standard port is equivalent to without", func(t *testing.T) {
		allowed := []string{"https://example.com:443"}
		assert.True(t, IsValidOrigin("https://example.com", allowed))
		assert.True(t, IsValidOrigin("https://example.com:443", allowed))
	})

	t.Run("http with port 80 is equivalent to without", func(t *testing.T) {
		allowed := []string{"http://example.com:80"}
		assert.True(t, IsValidOrigin("http://example.com", allowed))
		assert.True(t, IsValidOrigin("http://example.com:80", allowed))
	})

	// --- Custom ports ---

	t.Run("localhost with custom port", func(t *testing.T) {
		allowed := []string{"http://localhost:3000"}
		assert.True(t, IsValidOrigin("http://localhost:3000", allowed))
		assert.True(t, IsValidOrigin("http://localhost:3000/app", allowed))
		assert.False(t, IsValidOrigin("http://localhost:4000", allowed))
		assert.False(t, IsValidOrigin("http://localhost", allowed))
	})

	t.Run("domain with non-standard port", func(t *testing.T) {
		allowed := []string{"https://staging.example.com:8443"}
		assert.True(t, IsValidOrigin("https://staging.example.com:8443", allowed))
		assert.False(t, IsValidOrigin("https://staging.example.com", allowed))
		assert.False(t, IsValidOrigin("https://staging.example.com:9443", allowed))
	})

	// --- Allowed origins without protocol ---

	t.Run("allowed origin without protocol", func(t *testing.T) {
		allowed := []string{"example.com"}
		assert.True(t, IsValidOrigin("https://example.com", allowed))
		assert.True(t, IsValidOrigin("http://example.com", allowed))
		assert.False(t, IsValidOrigin("https://evil.com", allowed))
	})

	t.Run("allowed origin as host:port without protocol", func(t *testing.T) {
		allowed := []string{"localhost:3000"}
		assert.True(t, IsValidOrigin("http://localhost:3000", allowed))
		assert.True(t, IsValidOrigin("http://localhost:3000/callback", allowed))
		assert.False(t, IsValidOrigin("http://localhost:4000", allowed))
	})

	// --- Subdomain matching ---

	t.Run("subdomain is distinct from root domain", func(t *testing.T) {
		allowed := []string{"https://example.com"}
		assert.False(t, IsValidOrigin("https://api.example.com", allowed))
		assert.False(t, IsValidOrigin("https://auth.example.com", allowed))
	})

	t.Run("specific subdomain allowed", func(t *testing.T) {
		allowed := []string{"https://auth.example.com"}
		assert.True(t, IsValidOrigin("https://auth.example.com", allowed))
		assert.False(t, IsValidOrigin("https://example.com", allowed))
		assert.False(t, IsValidOrigin("https://api.example.com", allowed))
	})

	t.Run("deep subdomain", func(t *testing.T) {
		allowed := []string{"https://app.auth.example.com"}
		assert.True(t, IsValidOrigin("https://app.auth.example.com", allowed))
		assert.False(t, IsValidOrigin("https://auth.example.com", allowed))
		assert.False(t, IsValidOrigin("https://other.auth.example.com", allowed))
	})

	// --- Wildcard subdomain matching ---

	t.Run("wildcard subdomain", func(t *testing.T) {
		allowed := []string{"*.example.com"}
		assert.True(t, IsValidOrigin("https://auth.example.com", allowed))
		assert.True(t, IsValidOrigin("https://api.example.com", allowed))
		assert.True(t, IsValidOrigin("https://app.staging.example.com", allowed))
		assert.False(t, IsValidOrigin("https://example.com", allowed))
		assert.False(t, IsValidOrigin("https://evil.com", allowed))
		assert.False(t, IsValidOrigin("https://exampleXcom.evil.com", allowed))
	})

	t.Run("wildcard subdomain with port", func(t *testing.T) {
		allowed := []string{"*.example.com:8080"}
		assert.True(t, IsValidOrigin("https://api.example.com:8080", allowed))
		assert.False(t, IsValidOrigin("https://api.example.com", allowed))
		assert.False(t, IsValidOrigin("https://api.example.com:9090", allowed))
	})

	// --- Multiple allowed origins ---

	t.Run("multiple allowed origins", func(t *testing.T) {
		allowed := []string{
			"https://app.example.com",
			"https://admin.example.com",
			"http://localhost:3000",
		}
		assert.True(t, IsValidOrigin("https://app.example.com", allowed))
		assert.True(t, IsValidOrigin("https://admin.example.com", allowed))
		assert.True(t, IsValidOrigin("http://localhost:3000", allowed))
		assert.False(t, IsValidOrigin("https://evil.com", allowed))
		assert.False(t, IsValidOrigin("https://example.com", allowed))
	})

	// --- Attacker URLs (security cases) ---

	t.Run("attacker URL rejected", func(t *testing.T) {
		allowed := []string{"https://example.com"}
		assert.False(t, IsValidOrigin("https://attacker.com/steal", allowed))
		assert.False(t, IsValidOrigin("https://example.com.attacker.com", allowed))
		assert.False(t, IsValidOrigin("https://attacker.com?ref=example.com", allowed))
	})

	t.Run("attacker URL with similar name rejected", func(t *testing.T) {
		allowed := []string{"https://myapp.com"}
		assert.False(t, IsValidOrigin("https://notmyapp.com", allowed))
		assert.False(t, IsValidOrigin("https://myapp.com.evil.com", allowed))
		assert.False(t, IsValidOrigin("https://myapp.com.evil.com:3000", allowed))
	})

	// --- Live domain scenarios ---

	t.Run("production domain with www", func(t *testing.T) {
		allowed := []string{"https://www.myapp.com"}
		assert.True(t, IsValidOrigin("https://www.myapp.com", allowed))
		assert.False(t, IsValidOrigin("https://myapp.com", allowed))
	})

	t.Run("both www and apex allowed", func(t *testing.T) {
		allowed := []string{"https://myapp.com", "https://www.myapp.com"}
		assert.True(t, IsValidOrigin("https://myapp.com", allowed))
		assert.True(t, IsValidOrigin("https://www.myapp.com", allowed))
		assert.False(t, IsValidOrigin("https://api.myapp.com", allowed))
	})

	t.Run("wildcard with live domain", func(t *testing.T) {
		allowed := []string{"*.myapp.com"}
		assert.True(t, IsValidOrigin("https://www.myapp.com", allowed))
		assert.True(t, IsValidOrigin("https://api.myapp.com", allowed))
		assert.True(t, IsValidOrigin("https://staging.api.myapp.com", allowed))
		assert.False(t, IsValidOrigin("https://myapp.com", allowed))
		assert.False(t, IsValidOrigin("https://evil.com", allowed))
	})
}

func TestIsValidRedirectURI(t *testing.T) {
	hostname := "https://myserver.com"

	t.Run("wildcard config allows self-origin redirect", func(t *testing.T) {
		assert.True(t, IsValidRedirectURI("https://myserver.com/app/reset-password", []string{"*"}, hostname))
		assert.True(t, IsValidRedirectURI("https://myserver.com/callback", []string{"*"}, hostname))
		assert.True(t, IsValidRedirectURI("https://myserver.com", []string{"*"}, hostname))
	})

	t.Run("wildcard config rejects attacker URL", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("https://attacker.com/capture", []string{"*"}, hostname))
		assert.False(t, IsValidRedirectURI("https://evil.com/steal", []string{"*"}, hostname))
		assert.False(t, IsValidRedirectURI("https://myserver.com.attacker.com/cb", []string{"*"}, hostname))
	})

	t.Run("wildcard config rejects subdomain mismatch", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("https://sub.myserver.com/cb", []string{"*"}, hostname))
	})

	t.Run("empty config defaults to wildcard behavior", func(t *testing.T) {
		assert.True(t, IsValidRedirectURI("https://myserver.com/cb", []string{}, hostname))
		assert.True(t, IsValidRedirectURI("https://myserver.com/cb", nil, hostname))
		assert.False(t, IsValidRedirectURI("https://attacker.com", []string{}, hostname))
		assert.False(t, IsValidRedirectURI("https://attacker.com", nil, hostname))
	})

	t.Run("explicit origins valid redirect", func(t *testing.T) {
		allowed := []string{"https://app.example.com"}
		assert.True(t, IsValidRedirectURI("https://app.example.com/callback", allowed, hostname))
		assert.True(t, IsValidRedirectURI("https://app.example.com", allowed, hostname))
	})

	t.Run("explicit origins invalid redirect", func(t *testing.T) {
		allowed := []string{"https://app.example.com"}
		assert.False(t, IsValidRedirectURI("https://attacker.com", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://app.example.com.evil.com", allowed, hostname))
	})

	t.Run("wildcard subdomain origins", func(t *testing.T) {
		allowed := []string{"*.example.com"}
		assert.True(t, IsValidRedirectURI("https://api.example.com/cb", allowed, hostname))
		assert.True(t, IsValidRedirectURI("https://auth.example.com/login", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://example.com", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://evil.com", allowed, hostname))
	})

	t.Run("javascript scheme rejected", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("javascript:alert(1)", []string{"*"}, hostname))
	})

	t.Run("data scheme rejected", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("data:text/html,<h1>evil</h1>", []string{"*"}, hostname))
	})

	t.Run("ftp scheme rejected", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("ftp://evil.com/file", []string{"*"}, hostname))
	})

	t.Run("empty string rejected", func(t *testing.T) {
		assert.False(t, IsValidRedirectURI("", []string{"*"}, hostname))
	})

	t.Run("port matching with localhost", func(t *testing.T) {
		localhostHost := "http://localhost:3000"
		assert.True(t, IsValidRedirectURI("http://localhost:3000/cb", []string{"*"}, localhostHost))
		assert.False(t, IsValidRedirectURI("http://localhost:4000/cb", []string{"*"}, localhostHost))
		assert.False(t, IsValidRedirectURI("http://localhost/cb", []string{"*"}, localhostHost))
	})

	t.Run("standard port normalization", func(t *testing.T) {
		assert.True(t, IsValidRedirectURI("https://myserver.com:443/cb", []string{"*"}, hostname))
	})

	t.Run("path is ignored in origin matching", func(t *testing.T) {
		assert.True(t, IsValidRedirectURI("https://myserver.com/any/path/here", []string{"*"}, hostname))
		assert.True(t, IsValidRedirectURI("https://myserver.com/app/reset-password?foo=bar", []string{"*"}, hostname))
	})

	t.Run("multiple allowed origins", func(t *testing.T) {
		allowed := []string{"https://app.example.com", "http://localhost:3000"}
		assert.True(t, IsValidRedirectURI("https://app.example.com/cb", allowed, hostname))
		assert.True(t, IsValidRedirectURI("http://localhost:3000/cb", allowed, hostname))
		assert.False(t, IsValidRedirectURI("https://attacker.com", allowed, hostname))
	})
}

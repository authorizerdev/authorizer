package parsers

import (
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

package http_handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAuthorizeResumeTarget(t *testing.T) {
	const self = "https://auth.example.com"
	cases := []struct {
		name     string
		redirect string
		hostname string
		want     bool
	}{
		{"own /app", "https://auth.example.com/app?state=s&client_id=c", self, true},
		{"own /app/signup", "https://auth.example.com/app/signup?state=s", self, true},
		{"explicit default port", "https://auth.example.com:443/app", self, true},
		{"host is case-insensitive", "https://AUTH.example.com/app", self, true},
		{"--url with trailing slash", "https://auth.example.com/app", self + "/", true},
		{"http deployment", "http://localhost:8080/app", "http://localhost:8080", true},

		{"scheme downgrade", "http://auth.example.com/app", self, false},
		{"other port", "https://auth.example.com:8443/app", self, false},
		{"other host", "https://evil.example.com/app", self, false},
		{"lookalike subdomain", "https://auth.example.com.evil.net/app", self, false},
		{"other path", "https://auth.example.com/graphql", self, false},
		{"path below /app", "https://auth.example.com/app/../graphql", self, false},
		{"trailing slash path", "https://auth.example.com/app/", self, false},
		{"fragment", "https://auth.example.com/app#state=s", self, false},
		{"userinfo", "https://user@auth.example.com/app", self, false},
		{"relative", "/app", self, false},
		{"javascript scheme", "javascript:alert(1)", self, false},
		{"empty", "", self, false},
		{"empty hostname", "https://auth.example.com/app", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isAuthorizeResumeTarget(tc.redirect, tc.hostname))
		})
	}
}

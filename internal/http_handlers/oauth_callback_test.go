package http_handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseScopes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string returns empty slice",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single scope value",
			input:    "openid",
			expected: []string{"openid"},
		},
		{
			name:     "comma-delimited scopes",
			input:    "openid,email,profile",
			expected: []string{"openid", "email", "profile"},
		},
		{
			name:     "space-delimited scopes",
			input:    "openid email profile",
			expected: []string{"openid", "email", "profile"},
		},
		{
			name:     "mixed delimiters prefer comma",
			input:    "openid,email profile",
			expected: []string{"openid", "email profile"},
		},
		{
			name:     "two comma-separated scopes",
			input:    "openid,email",
			expected: []string{"openid", "email"},
		},
		{
			name:     "two space-separated scopes",
			input:    "openid email",
			expected: []string{"openid", "email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseScopes(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

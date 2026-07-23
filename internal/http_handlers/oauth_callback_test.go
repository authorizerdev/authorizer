package http_handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// REGRESSION: Apple only sends the `user` form field on the very first
// authorization for a given app; every subsequent login omits it entirely
// (documented Apple behavior — a one-time grant, not re-sent). Before this
// fix, an absent field made json.Unmarshal([]byte(""), ...) fail and the
// whole callback 400 out, rejecting every returning Apple user. A malformed
// non-empty field is still a real error and must still be rejected.
func TestParseAppleUserField(t *testing.T) {
	tests := []struct {
		name    string
		userRaw string
		want    *AppleUserInfo
		wantErr bool
	}{
		{
			name:    "absent field (returning-user login) succeeds with zero value",
			userRaw: "",
			want:    &AppleUserInfo{},
		},
		{
			name:    "valid json (first-time signup) parses normally",
			userRaw: `{"email":"a@example.com","name":{"firstName":"Ada","lastName":"Lovelace"}}`,
			want: &AppleUserInfo{
				Email: "a@example.com",
				Name: struct {
					FirstName string `json:"firstName"`
					LastName  string `json:"lastName"`
				}{FirstName: "Ada", LastName: "Lovelace"},
			},
		},
		{
			name:    "non-empty malformed json still errors",
			userRaw: `{"email":`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAppleUserField(tt.userRaw)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

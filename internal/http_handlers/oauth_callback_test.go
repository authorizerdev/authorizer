package http_handlers

import (
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/oauth"
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

const (
	appleTestClientID = "apple-client-id"
	appleTestKID      = "apple-key-1"
)

// newAppleMockServer stands up a single httptest.Server that plays the role
// of Apple's OAuth token endpoint AND its OIDC discovery/JWKS endpoints
// (mirroring how processAppleUserInfo resolves both off the same
// TestOAuthBaseURL mechanism — see internal/http_handlers/oauth_callback.go
// and internal/oauth/get_oauth_config.go). `claims` is signed with `key` as
// the id_token returned by /token, after this stamps in the server's own URL
// as `iss`. Key generation/JWKS-serving/RS256-signing reuse the helpers
// already established for OIDC id-token mocking in oauth_sso_verify_test.go
// (ssoGenKey/ssoJWKS/ssoSignRS256).
func newAppleMockServer(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	claims["iss"] = srv.URL
	idToken := ssoSignRS256(t, key, appleTestKID, claims)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 srv.URL,
			"authorization_endpoint": srv.URL + "/authorize",
			"token_endpoint":         srv.URL + "/token",
			"jwks_uri":               srv.URL + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ssoJWKS(&key.PublicKey, appleTestKID))
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "mock-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     idToken,
		})
	})

	return srv
}

// newAppleCallbackProvider builds a minimal *httpProvider wired to exchange
// codes and verify id_tokens against mockBase, matching what
// OAuthCallbackHandler's Apple branch does in production.
func newAppleCallbackProvider(t *testing.T, mockBase string) *httpProvider {
	t.Helper()
	config.TestOAuthMockBaseOverride = mockBase
	t.Cleanup(func() { config.TestOAuthMockBaseOverride = "" })
	logger := zerolog.Nop()
	cfg := &config.Config{
		Env:               constants.E2EEnv,
		AppleClientID:     appleTestClientID,
		AppleClientSecret: "apple-client-secret",
	}
	oauthProv, err := oauth.New(cfg, &oauth.Dependencies{Log: &logger})
	require.NoError(t, err)
	return &httpProvider{
		Config:       cfg,
		Dependencies: Dependencies{Log: &logger, OAuthProvider: oauthProv},
	}
}

func appleGinContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/oauth_callback/apple", nil)
	return c
}

// REGRESSION (core bug): a repeat Apple login sends no `user` field at all.
// processAppleUserInfo must still succeed, pulling the email from the
// verified id_token independent of the (here, zero-value) AppleUserInfo —
// proving the fix's premise that GivenName/FamilyName are the only fields
// that depend on the `user` form field, not Email.
func TestProcessAppleUserInfo_ReturningUserNoUserField_Succeeds(t *testing.T) {
	key := ssoGenKey(t)
	now := time.Now()
	claims := jwt.MapClaims{
		"aud":   appleTestClientID,
		"email": "returning-user@example.com",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	srv := newAppleMockServer(t, key, claims)

	h := newAppleCallbackProvider(t, srv.URL)
	c := appleGinContext()

	// Simulates the fixed OAuthCallbackHandler branch: no `user` field present.
	userField, perr := parseAppleUserField("")
	require.NoError(t, perr)

	user, err := h.processAppleUserInfo(c, "fake-oauth-code", userField)
	require.NoError(t, err, "a returning Apple user (no `user` field) must not be rejected")
	require.NotNil(t, user)
	require.NotNil(t, user.Email)
	assert.Equal(t, "returning-user@example.com", *user.Email, "email must come from the verified id_token, independent of the `user` field")
	require.NotNil(t, user.GivenName)
	require.NotNil(t, user.FamilyName)
	assert.Equal(t, "", *user.GivenName, "no `user` field means no given name to report")
	assert.Equal(t, "", *user.FamilyName, "no `user` field means no family name to report")
}

// Sanity counterpart: first-time signup still works exactly as before when
// the `user` field IS present and valid — proves the fix doesn't regress the
// existing first-authorization flow.
func TestProcessAppleUserInfo_FirstTimeSignupWithUserField_Succeeds(t *testing.T) {
	key := ssoGenKey(t)
	now := time.Now()
	claims := jwt.MapClaims{
		"aud":   appleTestClientID,
		"email": "first-time@example.com",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	srv := newAppleMockServer(t, key, claims)

	h := newAppleCallbackProvider(t, srv.URL)
	c := appleGinContext()

	userField, perr := parseAppleUserField(`{"email":"first-time@example.com","name":{"firstName":"Ada","lastName":"Lovelace"}}`)
	require.NoError(t, perr)

	user, err := h.processAppleUserInfo(c, "fake-oauth-code", userField)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.Email)
	assert.Equal(t, "first-time@example.com", *user.Email)
	require.NotNil(t, user.GivenName)
	require.NotNil(t, user.FamilyName)
	assert.Equal(t, "Ada", *user.GivenName)
	assert.Equal(t, "Lovelace", *user.FamilyName)
}

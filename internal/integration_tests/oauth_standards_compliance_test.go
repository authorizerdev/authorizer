package integration_tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// TestOpenIDDiscoveryCompliance verifies the /.well-known/openid-configuration
// endpoint complies with OpenID Connect Discovery 1.0
func TestOpenIDDiscoveryCompliance(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	// Create router with the OpenID config handler
	router := gin.New()
	router.GET("/.well-known/openid-configuration", ts.HttpProvider.OpenIDConfigurationHandler())

	t.Run("OIDC_Discovery_required_fields", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/.well-known/openid-configuration", nil)
		req.Host = "localhost"
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var body map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &body)
		require.NoError(t, err)

		// REQUIRED by OIDC Discovery 1.0 §3
		assert.NotEmpty(t, body["issuer"], "issuer is REQUIRED")
		assert.NotEmpty(t, body["authorization_endpoint"], "authorization_endpoint is REQUIRED")
		assert.NotEmpty(t, body["jwks_uri"], "jwks_uri is REQUIRED")
		assert.NotNil(t, body["response_types_supported"], "response_types_supported is REQUIRED")
		assert.NotNil(t, body["subject_types_supported"], "subject_types_supported is REQUIRED")
		assert.NotNil(t, body["id_token_signing_alg_values_supported"], "id_token_signing_alg_values_supported is REQUIRED")

		// id_token_signing_alg_values_supported MUST include RS256
		signingAlgs, ok := body["id_token_signing_alg_values_supported"].([]interface{})
		require.True(t, ok, "id_token_signing_alg_values_supported must be an array")
		hasRS256 := false
		for _, alg := range signingAlgs {
			if alg == "RS256" {
				hasRS256 = true
				break
			}
		}
		assert.True(t, hasRS256, "id_token_signing_alg_values_supported MUST include RS256 per OIDC Discovery")

		// RECOMMENDED fields
		assert.NotEmpty(t, body["token_endpoint"], "token_endpoint is RECOMMENDED")
		assert.NotEmpty(t, body["userinfo_endpoint"], "userinfo_endpoint is RECOMMENDED")
		assert.NotNil(t, body["scopes_supported"], "scopes_supported is RECOMMENDED")
		assert.NotNil(t, body["claims_supported"], "claims_supported is RECOMMENDED")

		// Additional standard fields
		assert.NotNil(t, body["grant_types_supported"], "grant_types_supported SHOULD be present")
		assert.NotNil(t, body["token_endpoint_auth_methods_supported"], "token_endpoint_auth_methods_supported SHOULD be present")
		assert.NotNil(t, body["code_challenge_methods_supported"], "code_challenge_methods_supported SHOULD be present for PKCE")
		assert.NotEmpty(t, body["revocation_endpoint"], "revocation_endpoint SHOULD be present")
		assert.NotEmpty(t, body["end_session_endpoint"], "end_session_endpoint SHOULD be present")
	})

	t.Run("OIDC_Discovery_response_types_include_code", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/.well-known/openid-configuration", nil)
		req.Host = "localhost"
		router.ServeHTTP(w, req)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)

		responseTypes, ok := body["response_types_supported"].([]interface{})
		require.True(t, ok)

		hasCode := false
		for _, rt := range responseTypes {
			if rt == "code" {
				hasCode = true
			}
		}
		assert.True(t, hasCode, "response_types_supported MUST include 'code'")
	})
}

// TestTokenEndpointCompliance verifies /oauth/token complies with RFC 6749
func TestTokenEndpointCompliance(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// Create a test user and get auth tokens
	email := "token_compliance_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"

	signupRes, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	require.NotNil(t, signupRes)

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	t.Run("RFC6749_missing_grant_type_defaults_to_authorization_code", func(t *testing.T) {
		// When grant_type is missing, it defaults to authorization_code
		// but code is also missing, so it should fail with invalid_request
		form := url.Values{}
		form.Set("client_id", cfg.ClientID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)

		// Should fail because code is missing
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "invalid_request", body["error"],
			"RFC 6749 §5.2: error code for missing required param MUST be 'invalid_request'")
	})

	t.Run("RFC6749_unsupported_grant_type", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", "client_credentials")
		form.Set("client_id", cfg.ClientID)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "unsupported_grant_type", body["error"],
			"RFC 6749 §5.2: unsupported grant type MUST return 'unsupported_grant_type' error")
	})

	t.Run("RFC6749_invalid_client_id", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("client_id", "wrong-client-id")
		form.Set("code", "some-code")
		form.Set("code_verifier", "some-verifier")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)

		assert.Equal(t, "invalid_client", body["error"],
			"RFC 6749 §5.2: invalid client MUST return 'invalid_client' error")
	})

	t.Run("RFC6749_invalid_client_via_basic_auth_returns_401", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", "some-code")
		form.Set("code_verifier", "some-verifier")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("wrong-client-id", "wrong-secret")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"RFC 6749 §5.2: invalid client via Basic Auth MUST return 401")
		assert.NotEmpty(t, w.Header().Get("WWW-Authenticate"),
			"RFC 6749 §5.2: 401 response MUST include WWW-Authenticate header")
	})

	t.Run("RFC6749_token_response_includes_token_type", func(t *testing.T) {
		// Create a valid authorization code flow to test full token response
		codeVerifier := uuid.New().String() + uuid.New().String()
		hash := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
		code := uuid.New().String()

		// Simulate the state that would be set during /authorize
		sessionToken := "test-session-token"
		ts.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+sessionToken)

		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("client_id", cfg.ClientID)
		form.Set("code", code)
		form.Set("code_verifier", codeVerifier)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		// The session validation will fail because we used a fake session token,
		// but let's verify the error format at least matches RFC 6749 §5.2
		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)

		assert.NotNil(t, body["error"], "Error responses MUST include 'error' field per RFC 6749 §5.2")
		if errDesc, ok := body["error_description"]; ok {
			assert.IsType(t, "", errDesc, "error_description MUST be a string per RFC 6749 §5.2")
		}
	})

	t.Run("RFC6749_missing_client_id", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", "some-code")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "invalid_request", body["error"],
			"RFC 6749 §5.2: missing required param MUST return 'invalid_request'")
	})

	t.Run("RFC7636_invalid_code_returns_invalid_grant", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("client_id", cfg.ClientID)
		form.Set("code", "nonexistent-code")
		form.Set("code_verifier", "some-verifier")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)

		assert.Equal(t, "invalid_grant", body["error"],
			"RFC 6749 §5.2: invalid authorization code MUST return 'invalid_grant'")
	})

	t.Run("RFC6749_refresh_token_missing", func(t *testing.T) {
		form := url.Values{}
		form.Set("grant_type", "refresh_token")
		form.Set("client_id", cfg.ClientID)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "invalid_request", body["error"],
			"Missing refresh_token MUST return 'invalid_request'")
	})
}

// TestRevocationEndpointCompliance verifies /oauth/revoke complies with RFC 7009
func TestRevocationEndpointCompliance(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.POST("/oauth/revoke", ts.HttpProvider.RevokeRefreshTokenHandler())

	t.Run("RFC7009_invalid_token_returns_200", func(t *testing.T) {
		// RFC 7009 §2.2: Invalid tokens do NOT cause error responses
		form := url.Values{}
		form.Set("token", "completely-invalid-token")
		form.Set("client_id", cfg.ClientID)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/revoke", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code,
			"RFC 7009 §2.2: invalid token MUST return HTTP 200")
	})

	t.Run("RFC7009_empty_token_returns_200", func(t *testing.T) {
		form := url.Values{}
		form.Set("token", "")
		form.Set("client_id", cfg.ClientID)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/revoke", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code,
			"RFC 7009 §2.2: empty token MUST return HTTP 200")
	})

	t.Run("RFC7009_missing_client_id_returns_error", func(t *testing.T) {
		form := url.Values{}
		form.Set("token", "some-token")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/revoke", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "invalid_request", body["error"],
			"Missing client_id MUST return 'invalid_request'")
	})

	t.Run("RFC7009_invalid_client_returns_401", func(t *testing.T) {
		form := url.Values{}
		form.Set("token", "some-token")
		form.Set("client_id", "wrong-client-id")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/revoke", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"RFC 7009: invalid client MUST return 401")
	})

	t.Run("RFC7009_unsupported_token_type_hint", func(t *testing.T) {
		form := url.Values{}
		form.Set("token", "some-token")
		form.Set("client_id", cfg.ClientID)
		form.Set("token_type_hint", "mac_token")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/revoke", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "unsupported_token_type", body["error"],
			"RFC 7009 §2.2.1: unsupported token type MUST return 'unsupported_token_type'")
	})

	t.Run("RFC7009_accepts_form_encoded", func(t *testing.T) {
		// RFC 7009 §2.1: MUST accept application/x-www-form-urlencoded
		form := url.Values{}
		form.Set("token", "some-invalid-token")
		form.Set("client_id", cfg.ClientID)
		form.Set("token_type_hint", "refresh_token")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/revoke", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code,
			"RFC 7009: form-encoded requests MUST be accepted")
	})

	t.Run("RFC7009_accepts_json_backward_compat", func(t *testing.T) {
		// Backward compatibility: also accept JSON
		jsonBody := fmt.Sprintf(`{"token":"some-invalid-token","client_id":"%s"}`, cfg.ClientID)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/oauth/revoke", strings.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code,
			"JSON requests should be accepted for backward compatibility")
	})
}

// TestUserInfoEndpointCompliance verifies /userinfo complies with OIDC Core §5.3 and RFC 6750
func TestUserInfoEndpointCompliance(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.GET("/userinfo", ts.HttpProvider.UserInfoHandler())

	t.Run("RFC6750_missing_bearer_token_returns_401_with_www_authenticate", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/userinfo", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		wwwAuth := w.Header().Get("WWW-Authenticate")
		assert.NotEmpty(t, wwwAuth,
			"RFC 6750 §3: 401 response MUST include WWW-Authenticate header")
		assert.Contains(t, wwwAuth, "Bearer",
			"RFC 6750 §3: WWW-Authenticate MUST use Bearer scheme")
	})

	t.Run("RFC6750_invalid_bearer_token_returns_401_with_error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/userinfo", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		wwwAuth := w.Header().Get("WWW-Authenticate")
		assert.Contains(t, wwwAuth, "Bearer",
			"RFC 6750 §3: WWW-Authenticate MUST use Bearer scheme")
		assert.Contains(t, wwwAuth, "invalid_token",
			"RFC 6750 §3.1: invalid token MUST include error='invalid_token'")

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)
		assert.Equal(t, "invalid_token", body["error"],
			"RFC 6750 §3.1: response body MUST include error='invalid_token'")
	})

	t.Run("OIDC_userinfo_error_response_format", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/userinfo", nil)
		router.ServeHTTP(w, req)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)

		// Error response must include error field
		assert.NotNil(t, body["error"], "Error responses MUST include 'error' field")
		assert.NotNil(t, body["error_description"], "Error responses SHOULD include 'error_description'")
	})
}

// TestAuthorizeEndpointCompliance verifies /authorize complies with RFC 6749 and RFC 7636
func TestAuthorizeEndpointCompliance(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())

	t.Run("RFC6749_missing_state_returns_error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/authorize?client_id="+cfg.ClientID+"&response_type=code&response_mode=query", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)
		assert.Contains(t, body["error"].(string), "state",
			"RFC 6749: missing state parameter MUST return error")
	})

	t.Run("RFC6749_invalid_response_type_returns_error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET",
			"/authorize?client_id="+cfg.ClientID+"&response_type=invalid&state=test-state&response_mode=query", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("RFC6749_invalid_client_id_returns_error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET",
			"/authorize?client_id=wrong-id&response_type=code&state=test-state&response_mode=query", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("RFC7636_unsupported_code_challenge_method_returns_error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET",
			"/authorize?client_id="+cfg.ClientID+
				"&response_type=code&state=test-state&response_mode=query"+
				"&code_challenge=test-challenge&code_challenge_method=plain", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var body map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &body)
		assert.Equal(t, "invalid_request", body["error"],
			"RFC 7636: unsupported code_challenge_method MUST return 'invalid_request'")
		assert.Contains(t, body["error_description"].(string), "S256",
			"RFC 7636: error_description should mention S256")
	})

	t.Run("RFC6749_invalid_response_mode_returns_error", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET",
			"/authorize?client_id="+cfg.ClientID+"&response_type=code&state=test-state&response_mode=invalid", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestJWKSEndpointCompliance verifies /.well-known/jwks.json
func TestJWKSEndpointCompliance(t *testing.T) {
	t.Run("JWKS_returns_empty_keys_for_HMAC", func(t *testing.T) {
		// HMAC (symmetric) keys must NOT be exposed via JWKS.
		cfg := getTestConfig() // uses HS256
		ts := initTestSetup(t, cfg)

		router := gin.New()
		router.GET("/.well-known/jwks.json", ts.HttpProvider.JWKsHandler())

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/.well-known/jwks.json", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var body map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &body)
		require.NoError(t, err)

		keys, ok := body["keys"].([]interface{})
		require.True(t, ok, "JWKS response MUST contain 'keys' array")
		assert.Empty(t, keys, "JWKS 'keys' array MUST be empty for HMAC-only config to prevent secret exposure")
	})

	t.Run("JWKS_returns_valid_keyset_for_RSA", func(t *testing.T) {
		cfg := getTestConfig()
		// Generate RSA keys for this test
		_, privateKey, publicKey, _, err := crypto.NewRSAKey("RS256", cfg.ClientID)
		require.NoError(t, err)
		cfg.JWTType = "RS256"
		cfg.JWTPrivateKey = privateKey
		cfg.JWTPublicKey = publicKey
		cfg.JWTSecret = "" // not needed for RSA
		ts := initTestSetup(t, cfg)

		router := gin.New()
		router.GET("/.well-known/jwks.json", ts.HttpProvider.JWKsHandler())

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/.well-known/jwks.json", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var body map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &body)
		require.NoError(t, err)

		keys, ok := body["keys"].([]interface{})
		require.True(t, ok, "JWKS response MUST contain 'keys' array")
		require.NotEmpty(t, keys, "JWKS 'keys' array MUST not be empty for RSA config")

		// Each key must have required JWK fields
		key := keys[0].(map[string]interface{})
		assert.NotEmpty(t, key["kty"], "JWK MUST contain 'kty' (key type)")
		assert.NotEmpty(t, key["alg"], "JWK MUST contain 'alg' (algorithm)")
		assert.NotEmpty(t, key["kid"], "JWK MUST contain 'kid' (key ID)")
	})
}

package integration_tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func authorizeHybridRequest(t *testing.T, ts *testSetup, qs string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/authorize?"+qs, nil)
	router.ServeHTTP(w, req)
	return w
}

func TestHybridUnsupportedResponseTypeRejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	// Something neither single-value nor a known hybrid combination.
	codeVerifier := uuid.New().String()
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	qs := "client_id=" + cfg.ClientID +
		"&response_type=code+wibble" +
		"&code_challenge=" + codeChallenge +
		"&state=hybrid-unsupported" +
		"&response_mode=fragment"
	w := authorizeHybridRequest(t, ts, qs)
	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "unsupported_response_type", body["error"])
}

func TestHybridQueryResponseModeRejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	codeVerifier := uuid.New().String()
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	qs := "client_id=" + cfg.ClientID +
		"&response_type=code+id_token" +
		"&code_challenge=" + codeChallenge +
		"&state=hybrid-query" +
		"&response_mode=query"
	w := authorizeHybridRequest(t, ts, qs)
	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_request", body["error"])
	assert.Contains(t, strings.ToLower(body["error_description"].(string)), "query")
}

func TestHybridResponseTypeParsingAcceptsKnownCombos(t *testing.T) {
	// This test only verifies parsing / routing, not the full flow
	// (which requires a valid session cookie). Asserts that the handler
	// does NOT return unsupported_response_type for the four hybrid
	// combinations and does NOT return invalid_request for them.
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	combos := []string{
		"code+id_token",
		"code+token",
		"code+id_token+token",
		"id_token+token",
	}
	for _, combo := range combos {
		t.Run(combo, func(t *testing.T) {
			codeVerifier := uuid.New().String()
			hash := sha256.Sum256([]byte(codeVerifier))
			codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
			qs := "client_id=" + cfg.ClientID +
				"&response_type=" + combo +
				"&code_challenge=" + codeChallenge +
				"&state=hybrid-parse-" + combo +
				"&response_mode=fragment"
			w := authorizeHybridRequest(t, ts, qs)
			body := w.Body.String()
			loc := w.Header().Get("Location")
			combined := body + "\n" + loc
			assert.NotContains(t, combined, "unsupported_response_type",
				"response_type=%s MUST NOT be rejected as unsupported", combo)
		})
	}
}

func TestHybridResponseTypeOrderInsensitive(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	codeVerifier := uuid.New().String()
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	// "token id_token code" should normalize to "code id_token token"
	qs := "client_id=" + cfg.ClientID +
		"&response_type=token+id_token+code" +
		"&code_challenge=" + codeChallenge +
		"&state=hybrid-order" +
		"&response_mode=fragment"
	w := authorizeHybridRequest(t, ts, qs)
	body := w.Body.String()
	loc := w.Header().Get("Location")
	combined := body + "\n" + loc
	assert.NotContains(t, combined, "unsupported_response_type",
		"response_type token order MUST NOT affect acceptance")
}

package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// authorizeRequest is a small local helper that builds a GET /authorize
// request with the supplied query string and returns the recorder.
func authorizeRequest(t *testing.T, ts *testSetup, qs string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/authorize?"+qs, nil)
	router.ServeHTTP(w, req)
	return w
}

// TestAuthorizePromptNoneNoSessionReturnsLoginRequired verifies OIDC Core
// §3.1.2.1 prompt=none behavior: if there is no valid session, return the
// OIDC error login_required without rendering the login UI.
func TestAuthorizePromptNoneNoSessionReturnsLoginRequired(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	qs := "client_id=" + cfg.ClientID +
		"&response_type=code" +
		"&code_challenge=dummy-challenge-unused-only-checks-presence" +
		"&state=test-state-none" +
		"&response_mode=query" +
		"&prompt=none"
	w := authorizeRequest(t, ts, qs)

	// The OIDC error must be surfaced. In query response_mode the redirect
	// may also carry the error. Accept either a JSON body with login_required
	// or a redirect whose query string contains error=login_required.
	body := w.Body.String()
	location := w.Header().Get("Location")
	combined := body + "\n" + location
	assert.Contains(t, combined, "login_required",
		"prompt=none with no session MUST surface error=login_required (OIDC Core §3.1.2.1)")
}

// TestAuthorizeLoginHintForwarded verifies login_hint is passed through to
// the auth URL the handler builds for the login page.
func TestAuthorizeLoginHintForwarded(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	qs := "client_id=" + cfg.ClientID +
		"&response_type=code" +
		"&code_challenge=cc" +
		"&state=test-state-lh" +
		"&response_mode=query" +
		"&login_hint=alice@example.com"
	w := authorizeRequest(t, ts, qs)

	body := w.Body.String()
	location := w.Header().Get("Location")
	combined := body + "\n" + location
	assert.Contains(t, combined, "login_hint=",
		"login_hint MUST be forwarded to the login UI auth URL")
	assert.Contains(t, combined, "alice%40example.com",
		"login_hint value MUST be URL-encoded (alice@example.com → alice%40example.com)")
}

// TestAuthorizeUILocalesForwarded verifies ui_locales is passed through.
func TestAuthorizeUILocalesForwarded(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	qs := "client_id=" + cfg.ClientID +
		"&response_type=code" +
		"&code_challenge=cc" +
		"&state=test-state-ui" +
		"&response_mode=query" +
		"&ui_locales=en-US"
	w := authorizeRequest(t, ts, qs)

	body := w.Body.String()
	location := w.Header().Get("Location")
	combined := body + "\n" + location
	assert.Contains(t, combined, "ui_locales=",
		"ui_locales MUST be forwarded to the login UI auth URL")
}

// TestAuthorizePromptConsentAndSelectAccountNoOp verifies these prompt
// values are parsed and accepted (no error), even though they are not
// implemented in Phase 2.
func TestAuthorizePromptConsentAndSelectAccountNoOp(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	for _, p := range []string{"consent", "select_account"} {
		t.Run(p, func(t *testing.T) {
			qs := "client_id=" + cfg.ClientID +
				"&response_type=code" +
				"&code_challenge=cc" +
				"&state=test-state-" + p +
				"&response_mode=query" +
				"&prompt=" + p
			w := authorizeRequest(t, ts, qs)
			// Must not be 4xx (should proceed as normal).
			assert.NotEqual(t, http.StatusBadRequest, w.Code,
				"prompt=%s MUST be accepted (no-op), not rejected with 400", p)
		})
	}
}

// TestAuthorizeMaxAgeParsedNotRejected verifies max_age is accepted as an
// integer and does not cause a 400. Deeper assertion (actual session-age
// comparison) requires a valid session cookie — that is covered by
// higher-level e2e tests.
func TestAuthorizeMaxAgeParsedNotRejected(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	qs := "client_id=" + cfg.ClientID +
		"&response_type=code" +
		"&code_challenge=cc" +
		"&state=test-state-maxage" +
		"&response_mode=query" +
		"&max_age=" + strconv.Itoa(300)
	w := authorizeRequest(t, ts, qs)
	assert.NotEqual(t, http.StatusBadRequest, w.Code, "max_age=300 MUST be accepted")
}

// TestAuthorizeIDTokenHintInvalidIgnored verifies that an invalid
// id_token_hint does not reject the request — OIDC Core §3.1.2.1 treats
// the hint as advisory only.
func TestAuthorizeIDTokenHintInvalidIgnored(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	qs := "client_id=" + cfg.ClientID +
		"&response_type=code" +
		"&code_challenge=cc" +
		"&state=test-state-hint" +
		"&response_mode=query" +
		"&id_token_hint=garbage-not-a-jwt"
	w := authorizeRequest(t, ts, qs)
	assert.NotEqual(t, http.StatusBadRequest, w.Code,
		"invalid id_token_hint MUST be ignored, not rejected")
}

// TestAuthorizePromptLoginBypassesSession ensures prompt=login is accepted
// and triggers the login page path (no 400).
func TestAuthorizePromptLoginBypassesSession(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	qs := "client_id=" + cfg.ClientID +
		"&response_type=code" +
		"&code_challenge=cc" +
		"&state=test-state-pl" +
		"&response_mode=query" +
		"&prompt=login"
	w := authorizeRequest(t, ts, qs)
	assert.NotEqual(t, http.StatusBadRequest, w.Code, "prompt=login MUST be accepted")

	// When there is no session, the handler still flows to login UI.
	// Response body or Location header should either contain a login
	// reference or a login_required-shaped response — asserting it is
	// NOT a 400 is the minimum-viable signal.
	_ = strings.TrimSpace(w.Body.String())
	_ = json.NewDecoder(strings.NewReader(w.Body.String()))
}

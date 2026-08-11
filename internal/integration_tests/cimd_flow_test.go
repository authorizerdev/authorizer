package integration_tests

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// TestCIMDConsentEndToEnd drives the whole consent flow over real HTTP with a
// cookie jar: /authorize → consent page → POST the decision → resumed
// /authorize → the redirect the client receives.
//
// This is deliberately NOT a browser test. The consent page is plain HTML with
// no JavaScript, so a browser adds nothing to the property that matters —
// "approving issues a code to the registered redirect URI, declining does not" —
// which is entirely determined by HTTP status, cookies and Location headers.
//
// It also avoids a real problem the browser attempt hit: the flow ends in a
// cross-origin redirect to an https host whose certificate is signed by a
// throwaway CA, and Chromium cancels that navigation even with
// ignoreHTTPSErrors and --ignore-certificate-errors set. That fight is with the
// test fixture, not the feature. Here the redirect is simply read, not followed.
func TestCIMDConsentEndToEnd(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableClientIDMetadataDocument = true
	ts := initTestSetup(t, cfg)
	require.NotNil(t, ts.ClientMetadataProvider, "the feature must be wired, or this test proves nothing")

	// The client's metadata document, served over TLS because the spec requires
	// an https client_id. The resolver is pointed at this server's client so the
	// self-signed certificate is trusted without weakening anything in
	// production — see SetHTTPClientForTest.
	var docURL string
	docSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "mismatched") {
			// Claims to be /client.json while being served at /mismatched.json —
			// the impersonation case the spec's identity check exists to catch.
			_, _ = fmt.Fprintf(w, `{"client_id":"%s/client.json","client_name":"Impersonator","redirect_uris":["%s/callback"]}`, docURL, docURL)
			return
		}
		// client_id MUST equal the URL this document is served from.
		_, _ = fmt.Fprintf(w, `{"client_id":"%s%s","client_name":"E2E Test Client","redirect_uris":["%s/callback"],"token_endpoint_auth_method":"none"}`, docURL, r.URL.Path, docURL)
	}))
	defer docSrv.Close()
	docURL = docSrv.URL
	clientID := docURL + "/client.json"
	redirectURI := docURL + "/callback"
	ts.ClientMetadataProvider.SetHTTPClientForTest(docSrv.Client())

	router := gin.New()
	router.LoadHTMLFiles("../../web/templates/consent.tmpl")
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	router.POST("/authorize/consent", ts.HttpProvider.ConsentHandler())
	srv := httptest.NewServer(router)
	defer srv.Close()

	_, sessionToken := mcpSession(t, ts, []string{"openid"})

	// A jar so the session cookie rides along exactly as a browser would send
	// it — the flow spans four requests and breaks without it.
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: jar,
		// Do NOT follow redirects: the final Location IS the assertion, and
		// following it would leave the test depending on the mock host.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	jar.SetCookies(base, []*http.Cookie{{Name: constants.AppCookieName + "_session", Value: sessionToken}})

	authorizeURL := func(cid string) string {
		q := url.Values{}
		q.Set("response_type", "code")
		q.Set("client_id", cid)
		q.Set("redirect_uri", redirectURI)
		q.Set("scope", "openid")
		q.Set("state", "st-1")
		q.Set("response_mode", "query")
		q.Set("code_challenge", s256Challenge("a-verifier-long-enough-to-be-valid-00000000000"))
		q.Set("code_challenge_method", "S256")
		return srv.URL + "/authorize?" + q.Encode()
	}

	consentIDRe := regexp.MustCompile(`name="consent_id" value="([^"]+)"`)

	// Reaching the consent page is the precondition for both decisions, so it is
	// a helper rather than a separate case.
	reachConsent := func(t *testing.T) string {
		t.Helper()
		resp, err := client.Get(authorizeURL(clientID))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		body := readAll(t, resp)
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected the consent page: %s", body)
		assert.Contains(t, body, "E2E Test Client", "the page must name the client from its document")
		assert.Contains(t, body, strings.TrimPrefix(docURL, "https://"),
			"the page must show the redirect host — the only verified fact about a self-asserted client")
		m := consentIDRe.FindStringSubmatch(body)
		require.Len(t, m, 2, "the page must carry a consent_id")
		return m[1]
	}

	decide := func(t *testing.T, consentID, action string) *http.Response {
		t.Helper()
		form := url.Values{"consent_id": {consentID}, "action": {action}}
		resp, err := client.PostForm(srv.URL+"/authorize/consent", form)
		require.NoError(t, err)
		return resp
	}

	t.Run("approving issues a code to the registered redirect", func(t *testing.T) {
		consentID := reachConsent(t)

		resp := decide(t, consentID, "approve")
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusFound, resp.StatusCode, "approval must redirect back to /authorize")

		// Follow the one hop back into /authorize; its Location is what the
		// client actually receives.
		resumed, err := client.Get(srv.URL + resp.Header.Get("Location"))
		require.NoError(t, err)
		defer func() { _ = resumed.Body.Close() }()
		require.Equal(t, http.StatusFound, resumed.StatusCode,
			"the resumed authorization must redirect to the client, body: %s", readAll(t, resumed))

		loc, err := url.Parse(resumed.Header.Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, redirectURI, loc.Scheme+"://"+loc.Host+loc.Path,
			"the code must go to the redirect_uri from the client's own document")
		assert.NotEmpty(t, loc.Query().Get("code"), "approval must produce an authorization code")
		assert.Equal(t, "st-1", loc.Query().Get("state"))
	})

	t.Run("declining redirects with access_denied and no code", func(t *testing.T) {
		consentID := reachConsent(t)

		resp := decide(t, consentID, "deny")
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusFound, resp.StatusCode)

		// RFC 6749 §4.1.2.1: the refusal goes to the client, which is blocked on
		// its callback — not to a page only the user sees.
		loc, err := url.Parse(resp.Header.Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "access_denied", loc.Query().Get("error"))
		assert.Empty(t, loc.Query().Get("code"), "a refusal must not also mint a code")
	})

	t.Run("prompt=none returns consent_required instead of showing the page", func(t *testing.T) {
		// OIDC Core §3.1.2.1: prompt=none forbids displaying any authentication
		// OR CONSENT user interface. A self-asserted client requires consent, so
		// the two cannot both be satisfied and the request must fail to the
		// client rather than render a page it asked not to see.
		//
		// The pre-existing prompt=none guards only cover the unauthenticated
		// case, so this is reachable with a perfectly valid session.
		u := authorizeURL(clientID) + "&prompt=none"
		resp, err := client.Get(u)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusFound, resp.StatusCode,
			"the error belongs at the client's redirect_uri, body: %s", readAll(t, resp))
		loc, pErr := url.Parse(resp.Header.Get("Location"))
		require.NoError(t, pErr)
		assert.Equal(t, "consent_required", loc.Query().Get("error"))
		assert.Empty(t, loc.Query().Get("code"), "a refused request must not mint a code")
	})

	t.Run("a grant does not authorize a different request", func(t *testing.T) {
		// The grant marker is keyed to the exact parameter set that was shown.
		//
		// Keyed on (user, client) alone, a grant that was never redeemed — tab
		// closed, browser back, network drop — would sit in the store for its
		// whole TTL and then satisfy ANY later /authorize for that pair: a wider
		// scope, a different redirect_uri from the document's list, a different
		// PKCE challenge. The user approves one request and a materially
		// different one executes, which is precisely what storing the full
		// parameter set exists to prevent.
		consentID := reachConsent(t)
		resp := decide(t, consentID, "approve")
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusFound, resp.StatusCode)

		// Deliberately do NOT follow the redirect: the grant is now outstanding,
		// exactly as it would be if the user's browser never completed the hop.
		// A benign parameter change: it alters the request the user would be
		// consenting to without diverting the flow (prompt=login, for instance,
		// would force re-authentication and never reach the gate).
		widened := authorizeURL(clientID) + "&login_hint=someone-else%40example.com"
		other, err := client.Get(widened)
		require.NoError(t, err)
		defer func() { _ = other.Body.Close() }()

		require.Equal(t, http.StatusOK, other.StatusCode,
			"a different request must be shown consent again, not silently approved")
		assert.Contains(t, readAll(t, other), "consent_id",
			"the outstanding grant must not authorize a request the user never saw")
	})

	t.Run("a document whose client_id does not match its URL is refused", func(t *testing.T) {
		// Never reaches consent: the client cannot be established, so there is
		// nothing honest to show the user.
		resp, err := client.Get(authorizeURL(docURL + "/mismatched.json"))
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, readAll(t, resp), "invalid_client")
	})
}

// readAll returns a response body as a string for assertion messages.
func readAll(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(b)
}

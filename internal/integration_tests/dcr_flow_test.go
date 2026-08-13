package integration_tests

import (
	"encoding/json"
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

const dcrVerifier = "a-verifier-long-enough-to-be-valid-00000000000"

// dcrRouter mounts the routes a dynamically registered client actually uses.
func dcrRouter(ts *testSetup) *gin.Engine {
	router := gin.New()
	router.LoadHTMLFiles("../../web/templates/consent.tmpl")
	router.POST("/oauth/register", ts.HttpProvider.RegisterClientHandler())
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	router.POST("/authorize/consent", ts.HttpProvider.ConsentHandler())
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())
	return router
}

// registerClient POSTs an RFC 7591 registration and returns the decoded body.
func registerClient(t *testing.T, router http.Handler, body string) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

// TestDynamicClientRegistrationEndToEnd drives the whole RFC 7591 path: a client
// registers itself, is sent through consent because nobody vouched for it, and
// redeems the resulting code as a public client using PKCE alone.
//
// The point of running it end to end rather than per-handler is that the parts
// only work together: registration writes a row whose Kind drives the consent
// gate, and whose empty secret is what makes the PKCE-only token exchange legal.
func TestDynamicClientRegistrationEndToEnd(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableDynamicClientRegistration = true
	ts := initTestSetup(t, cfg)

	router := dcrRouter(ts)
	srv := httptest.NewServer(router)
	defer srv.Close()

	redirectURI := "http://127.0.0.1:5599/callback"

	code, body := registerClient(t, router,
		`{"client_name":"Dynamic Test Client","redirect_uris":["`+redirectURI+`"],"token_endpoint_auth_method":"none"}`)
	require.Equal(t, http.StatusCreated, code, "registration must return 201 Created: %v", body)
	clientID, _ := body["client_id"].(string)
	require.NotEmpty(t, clientID, "registration must issue a client_id")

	t.Run("the response is a public client and carries no secret", func(t *testing.T) {
		// RFC 7591 §3.2.1. A secret here would mean an anonymous caller had just
		// created a CONFIDENTIAL client, which is the thing this endpoint must
		// never do.
		assert.Equal(t, "none", body["token_endpoint_auth_method"])
		assert.NotContains(t, body, "client_secret")
		assert.NotContains(t, body, "client_secret_expires_at")
		assert.NotEmpty(t, body["client_id_issued_at"])
		assert.NotEqual(t, "Dynamic Test Client", clientID,
			"the client_id must be server-generated, never derived from caller-supplied metadata")
	})

	t.Run("the registered client is stored as self-registered", func(t *testing.T) {
		// The Kind is what routes this client through consent later; if it were
		// stored as an ordinary interactive client the consent gate would be
		// silently skipped and the rest of this test would still pass.
		stored, err := ts.StorageProvider.GetClientByClientID(t.Context(), clientID)
		require.NoError(t, err)
		require.NotNil(t, stored)
		assert.Equal(t, constants.ClientKindDynamic, stored.Kind)
		assert.Empty(t, stored.ClientSecret, "a dynamically registered client must be public")
		assert.True(t, stored.IsActive)
	})

	authorizeQuery := func() url.Values {
		q := url.Values{}
		q.Set("response_type", "code")
		q.Set("client_id", clientID)
		q.Set("redirect_uri", redirectURI)
		q.Set("scope", "openid")
		q.Set("state", "dcr-state")
		q.Set("response_mode", "query")
		q.Set("code_challenge", s256Challenge(dcrVerifier))
		q.Set("code_challenge_method", "S256")
		return q
	}

	_, sessionToken := mcpSession(t, ts, []string{"openid"})

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	jar.SetCookies(base, []*http.Cookie{{Name: constants.AppCookieName + "_session", Value: sessionToken}})

	consentIDRe := regexp.MustCompile(`name="consent_id" value="([^"]+)"`)

	t.Run("a self-registered client is sent through consent", func(t *testing.T) {
		// RFC 7591 §5: "a rogue client might use the name and logo of a
		// legitimate client", so servers should "present warning messages to
		// end-users about dynamically registered clients". The name below was
		// chosen by the caller and verified by nobody.
		resp, gErr := client.Get(srv.URL + "/authorize?" + authorizeQuery().Encode())
		require.NoError(t, gErr)
		defer func() { _ = resp.Body.Close() }()
		page := readAll(t, resp)
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected the consent page: %s", page)
		assert.Contains(t, page, "Dynamic Test Client", "the page must name the client it registered as")
		assert.Contains(t, page, "127.0.0.1:5599",
			"the page must show the redirect host — the only verified fact about a self-asserted client")
		assert.Contains(t, page, "runs on your own computer",
			"a loopback-only client must carry the impersonation warning")

		// The page's CSP must permit the form submission to end at the client's
		// redirect_uri. This is not a hardening nicety — it is what makes the
		// button work at all.
		//
		// Approving POSTs to /authorize/consent, which redirects to /authorize,
		// which redirects to the client's redirect_uri. A browser enforces
		// form-action across that WHOLE redirect chain, so the default
		// `form-action 'self'` aborts the navigation at the last hop: the user
		// stays on the consent page, assumes the click missed, clicks again, and
		// the second POST fails with "expired or already used" because the first
		// one already consumed the consent. That is precisely what a real user
		// hit, and no test caught it because every automated check either drove
		// the redirects itself or asserted the Location header instead of
		// letting a browser follow it.
		csp := resp.Header.Get("Content-Security-Policy")
		require.NotEmpty(t, csp, "the consent page must still carry a CSP")
		assert.Contains(t, csp, "http://127.0.0.1:5599",
			"form-action must allow the redirect_uri being approved, or the browser blocks the redirect after approval")
		assert.NotContains(t, csp, "form-action 'self';",
			"form-action 'self' alone silently breaks approval for any off-origin redirect_uri")
	})

	t.Run("approving issues a code redeemable with PKCE and no secret", func(t *testing.T) {
		resp, gErr := client.Get(srv.URL + "/authorize?" + authorizeQuery().Encode())
		require.NoError(t, gErr)
		defer func() { _ = resp.Body.Close() }()
		m := consentIDRe.FindStringSubmatch(readAll(t, resp))
		require.Len(t, m, 2, "the page must carry a consent_id")

		approved, pErr := client.PostForm(srv.URL+"/authorize/consent",
			url.Values{"consent_id": {m[1]}, "action": {"approve"}})
		require.NoError(t, pErr)
		defer func() { _ = approved.Body.Close() }()
		require.Equal(t, http.StatusFound, approved.StatusCode)

		resumed, rErr := client.Get(srv.URL + approved.Header.Get("Location"))
		require.NoError(t, rErr)
		defer func() { _ = resumed.Body.Close() }()
		require.Equal(t, http.StatusFound, resumed.StatusCode,
			"the resumed authorization must redirect to the client: %s", readAll(t, resumed))

		loc, lErr := url.Parse(resumed.Header.Get("Location"))
		require.NoError(t, lErr)
		authCode := loc.Query().Get("code")
		require.NotEmpty(t, authCode, "approval must produce an authorization code")

		// The exchange that proves the whole point: no client_secret anywhere,
		// PKCE alone binding the code to the instance that started the flow.
		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {authCode},
			"redirect_uri":  {redirectURI},
			"client_id":     {clientID},
			"code_verifier": {dcrVerifier},
		}
		tokenResp, tErr := http.Post(srv.URL+"/oauth/token",
			"application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		require.NoError(t, tErr)
		defer func() { _ = tokenResp.Body.Close() }()
		tokenBody := readAll(t, tokenResp)
		require.Equal(t, http.StatusOK, tokenResp.StatusCode,
			"a public client must redeem its code with PKCE alone: %s", tokenBody)
		assert.Contains(t, tokenBody, "access_token")
	})

	t.Run("PKCE is required, not optional", func(t *testing.T) {
		// RFC 9700 §2.1.1 "Public clients MUST use PKCE"; OAuth 2.1 §4.1.1 has
		// the authorization server MUST enforce it. Rejected at /authorize rather
		// than at the token endpoint so the user is never asked to log in and
		// approve a request that was never completable.
		q := authorizeQuery()
		q.Del("code_challenge")
		q.Del("code_challenge_method")
		w := doAuthorizeGET(router, q, sessionToken)
		require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		assert.Contains(t, w.Body.String(), "code_challenge is required")
	})

	t.Run("a public client can refresh with client_id alone", func(t *testing.T) {
		// Not hypothetical: Claude Code registers grant_types
		// ["authorization_code","refresh_token"], so this path runs on every
		// long-lived connection. RFC 6749 §6 has a public client refresh with its
		// client_id and no secret, and the resource binding must survive the
		// rotation — an earlier bug in this area dropped the RFC 8707 resource on
		// refresh, which killed MCP connections at the first token rotation
		// rather than at connect time.
		q := authorizeQuery()
		q.Set("scope", "openid offline_access")
		q.Set("resource", "http://localhost:8099/mcp")
		q.Set("state", "dcr-refresh")

		// Consent first — a self-registered client always passes through it.
		page, gErr := client.Get(srv.URL + "/authorize?" + q.Encode())
		require.NoError(t, gErr)
		defer func() { _ = page.Body.Close() }()
		m := consentIDRe.FindStringSubmatch(readAll(t, page))
		require.Len(t, m, 2)

		approved, pErr := client.PostForm(srv.URL+"/authorize/consent",
			url.Values{"consent_id": {m[1]}, "action": {"approve"}})
		require.NoError(t, pErr)
		defer func() { _ = approved.Body.Close() }()
		resumed, rErr := client.Get(srv.URL + approved.Header.Get("Location"))
		require.NoError(t, rErr)
		defer func() { _ = resumed.Body.Close() }()
		loc, lErr := url.Parse(resumed.Header.Get("Location"))
		require.NoError(t, lErr)
		authCode := loc.Query().Get("code")
		require.NotEmpty(t, authCode, "body: %s", readAll(t, resumed))

		exchange := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {authCode},
			"redirect_uri":  {redirectURI},
			"client_id":     {clientID},
			"code_verifier": {dcrVerifier},
			"resource":      {"http://localhost:8099/mcp"},
		}
		tr, tErr := http.Post(srv.URL+"/oauth/token",
			"application/x-www-form-urlencoded", strings.NewReader(exchange.Encode()))
		require.NoError(t, tErr)
		defer func() { _ = tr.Body.Close() }()
		var tok map[string]any
		require.NoError(t, json.Unmarshal([]byte(readAll(t, tr)), &tok))
		require.Equal(t, http.StatusOK, tr.StatusCode, "body: %v", tok)
		refreshToken, _ := tok["refresh_token"].(string)
		require.NotEmpty(t, refreshToken, "offline_access must yield a refresh token, or this test proves nothing")

		// The refresh itself: client_id only, no secret.
		refresh := url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {refreshToken},
			"client_id":     {clientID},
		}
		rr, rfErr := http.Post(srv.URL+"/oauth/token",
			"application/x-www-form-urlencoded", strings.NewReader(refresh.Encode()))
		require.NoError(t, rfErr)
		defer func() { _ = rr.Body.Close() }()
		var rotated map[string]any
		require.NoError(t, json.Unmarshal([]byte(readAll(t, rr)), &rotated))
		require.Equal(t, http.StatusOK, rr.StatusCode,
			"a public client must refresh with client_id alone: %v", rotated)

		access, _ := rotated["access_token"].(string)
		require.NotEmpty(t, access)
		claims, cErr := ts.TokenProvider.ParseJWTToken(access)
		require.NoError(t, cErr)
		assert.Equal(t, "http://localhost:8099/mcp", claims["aud"],
			"the rotated token must stay bound to the resource the user consented to")
	})

	t.Run("an implicit response type is refused", func(t *testing.T) {
		// OAuth 2.1 removes the implicit grant and MCP mandates OAuth 2.1.
		// Independently: implicit delivers a bearer token into the URL fragment
		// with nothing binding it to the requester, which for a client nobody
		// vouched for is the worst available combination. It is also what the
		// client itself registered — response_types ["code"] — so this is
		// enforcement, not a new restriction.
		q := authorizeQuery()
		q.Set("response_type", "id_token token")
		q.Set("response_mode", "fragment")
		w := doAuthorizeGET(router, q, sessionToken)
		require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		assert.Contains(t, w.Body.String(), "unsupported_response_type")
	})

	t.Run("PKCE plain is refused even outside strict mode", func(t *testing.T) {
		// "plain" carries the verifier in the same request as the challenge, so
		// it protects nothing against an attacker who can read that request —
		// which is exactly the threat model for a client with no secret.
		require.False(t, cfg.OAuth21Strict, "this case is only meaningful with strict mode off")
		q := authorizeQuery()
		q.Set("code_challenge", "a-plain-challenge-value-000000000000000000000")
		q.Set("code_challenge_method", "plain")
		w := doAuthorizeGET(router, q, sessionToken)
		require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
		assert.Contains(t, w.Body.String(), "must be S256")
	})
}

// TestDynamicClientRegistrationRefusals pins what the endpoint will not accept.
// Each case is a way an anonymous caller could otherwise widen what it gets.
func TestDynamicClientRegistrationRefusals(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableDynamicClientRegistration = true
	ts := initTestSetup(t, cfg)
	router := dcrRouter(ts)

	cases := []struct {
		name, body, wantError, why string
	}{
		{
			name:      "a confidential client may not self-register",
			body:      `{"client_name":"c","redirect_uris":["https://app.example.com/cb"],"token_endpoint_auth_method":"client_secret_basic"}`,
			wantError: "invalid_client_metadata",
			why:       "issuing a secret to an anonymous caller creates a confidential client nobody vouched for",
		},
		{
			name:      "the RFC default auth method is refused, not honoured",
			body:      `{"client_name":"c","redirect_uris":["https://app.example.com/cb"],"grant_types":["authorization_code"]}`,
			wantError: "",
			why:       "RFC 7591 §2 defaults the omitted field to client_secret_basic; omitted must mean public here",
		},
		{
			name:      "client_credentials may not be registered",
			body:      `{"client_name":"c","redirect_uris":["https://app.example.com/cb"],"grant_types":["client_credentials"]}`,
			wantError: "invalid_client_metadata",
			why:       "it issues a token with no user and no consent",
		},
		{
			name:      "redirect_uris is required",
			body:      `{"client_name":"c"}`,
			wantError: "invalid_redirect_uri",
			why:       "every grant accepted here is redirect-based",
		},
		{
			name:      "a non-loopback http redirect is refused",
			body:      `{"client_name":"c","redirect_uris":["http://app.example.com/cb"]}`,
			wantError: "invalid_redirect_uri",
			why:       "MCP requires redirect URIs to be localhost or https; otherwise codes cross the network in the clear",
		},
		{
			name:      "a custom scheme is refused",
			body:      `{"client_name":"c","redirect_uris":["myapp://callback"]}`,
			wantError: "invalid_redirect_uri",
			why:       "the server cannot tell which local application the OS will hand the code to",
		},
		{
			name:      "a fragment is refused",
			body:      `{"client_name":"c","redirect_uris":["https://app.example.com/cb#x"]}`,
			wantError: "invalid_redirect_uri",
			why:       "RFC 6749 §3.1.2: the endpoint URI MUST NOT include a fragment",
		},
		{
			name:      "a comma is refused",
			body:      `{"client_name":"c","redirect_uris":["https://app.example.com/cb?a=1,2"]}`,
			wantError: "invalid_redirect_uri",
			why:       "comma is the storage encoding for the list, so one would split into two registered URIs",
		},
		{
			name:      "a non-code response type is refused",
			body:      `{"client_name":"c","redirect_uris":["https://app.example.com/cb"],"response_types":["token"]}`,
			wantError: "invalid_client_metadata",
			why:       "silently narrowing to code would surface only when the client's callback got something it cannot parse",
		},
		{
			name:      "a non-object body is refused",
			body:      `"not-an-object"`,
			wantError: "invalid_client_metadata",
			why:       "the endpoint is unauthenticated; it must not fall through on malformed input",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := registerClient(t, router, tc.body)
			if tc.wantError == "" {
				// The "omitted auth method" case: accepted, but pinned to public.
				require.Equal(t, http.StatusCreated, status, "%s — body: %v", tc.why, body)
				assert.Equal(t, "none", body["token_endpoint_auth_method"], tc.why)
				return
			}
			require.Equal(t, http.StatusBadRequest, status, "%s — body: %v", tc.why, body)
			assert.Equal(t, tc.wantError, body["error"], tc.why)
		})
	}
}

// TestDynamicClientRegistrationDisabled pins that the feature is inert when off.
func TestDynamicClientRegistrationDisabled(t *testing.T) {
	cfg := getTestConfig()
	// Deliberately NOT setting EnableDynamicClientRegistration.
	ts := initTestSetup(t, cfg)

	t.Run("the handler refuses even if it is somehow routed", func(t *testing.T) {
		// The route is only mounted when the flag is on, so this is defence in
		// depth. It is asserted because an unauthenticated write endpoint is the
		// wrong place to depend on registration order staying correct.
		router := gin.New()
		router.POST("/oauth/register", ts.HttpProvider.RegisterClientHandler())
		status, body := registerClient(t, router, `{"redirect_uris":["https://app.example.com/cb"]}`)
		assert.Equal(t, http.StatusNotFound, status, "body: %v", body)
	})

	t.Run("registration_endpoint is not advertised", func(t *testing.T) {
		// RFC 8414 §2 treats an omitted field as "not supported". Advertising an
		// endpoint that is not mounted would send clients into a 404 at the one
		// step they cannot recover from.
		w := httptest.NewRecorder()
		r := gin.New()
		r.GET("/.well-known/openid-configuration", ts.HttpProvider.OpenIDConfigurationHandler())
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil))
		require.Equal(t, http.StatusOK, w.Code)

		var meta map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &meta))
		assert.NotContains(t, meta, "registration_endpoint")
	})
}

// TestDynamicClientRegistrationAdvertised is the other half of the pair above:
// when enabled, discovery must point at the endpoint that is actually mounted.
func TestDynamicClientRegistrationAdvertised(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableDynamicClientRegistration = true
	ts := initTestSetup(t, cfg)

	w := httptest.NewRecorder()
	r := gin.New()
	r.GET("/.well-known/openid-configuration", ts.HttpProvider.OpenIDConfigurationHandler())
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var meta map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &meta))
	// This exact field is what a client looks for before it will attempt DCR;
	// its absence is the "does not support dynamic client registration" refusal.
	assert.Contains(t, meta, "registration_endpoint")
	endpoint, _ := meta["registration_endpoint"].(string)
	assert.True(t, strings.HasSuffix(endpoint, "/oauth/register"), "got %q", endpoint)
}

// TestOperatorRegisteredClientIsUnaffected is the backward-compatibility guard.
//
// Everything added for self-registered clients — the consent interstitial and
// mandatory S256 PKCE — must be invisible to clients that already worked. An
// operator-created client was vouched for by a human, which is the entire basis
// for not interrupting its users, and it may legitimately predate PKCE.
func TestOperatorRegisteredClientIsUnaffected(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableDynamicClientRegistration = true
	ts := initTestSetup(t, cfg)

	// The deployment's own reserved client: interactive, operator-configured.
	_, sessionToken := mcpSession(t, ts, []string{"openid"})
	router := dcrRouter(ts)

	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", cfg.ClientID)
	// The reserved client registers no redirect URIs of its own, so it falls
	// back to the AllowedOrigins allow-list — which is exactly the legacy path
	// that must not regress.
	q.Set("redirect_uri", "http://localhost:3000/callback")
	q.Set("scope", "openid")
	q.Set("state", "bc-state")
	q.Set("response_mode", "query")
	// No code_challenge at all — the case that must keep working.

	w := doAuthorizeGET(router, q, sessionToken)
	require.Equal(t, http.StatusFound, w.Code,
		"an operator-registered client must still complete without PKCE and without consent: %s", w.Body.String())
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	assert.NotEmpty(t, loc.Query().Get("code"), "the pre-existing flow must still mint a code")
	assert.NotContains(t, w.Body.String(), "consent_id",
		"an operator-registered client must never be sent through the consent screen")
}

package integration_tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
)

// TestConsentDelegatedTokenBoundsAndBypass settles the two disputed facts in
// GHSA-hhhh-p743-q3cp against current main.
//
// The client is RFC 7591 self-registered (ClientKindDynamic) rather than a CIMD
// URL on purpose: a self-registered client resolves from the registry with no
// outbound fetch, so /authorize reaches the session and consent-grant checks.
// A CIMD client_id that does not serve a document is refused with
// invalid_client BEFORE either check — which makes any conclusion drawn from it
// about the session worthless.
func TestConsentDelegatedTokenBoundsAndBypass(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	// The victim.
	email := "consent_deleg_" + uuid.NewString() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)
	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	// A third-party client that registered ITSELF — the consent gate exists
	// because nobody at this deployment vouched for it.
	const redirectURI = "http://localhost:3000/cb"
	thirdParty, err := ts.StorageProvider.AddClient(context.Background(), &schemas.Client{
		Name:          "third-party-" + uuid.NewString(),
		Kind:          constants.ClientKindDynamic,
		RedirectURIs:  redirectURI,
		AllowedScopes: "openid,profile,email",
		GrantTypes:    "authorization_code,refresh_token",
		IsActive:      true,
	})
	require.NoError(t, err)

	// An agent whose ceiling is `openid` ONLY — the minimal delegation.
	agentID, _ := newDelegationAgent(t, ts, "openid")
	delegated := mintDelegated(t, ts, user.ID, agentID, testAuthorizerHost(ts))

	router := gin.New()
	router.LoadHTMLFiles(
		"../../web/templates/consent.tmpl",
		"../../web/templates/consent_error.tmpl",
		"../../web/templates/au_shell.tmpl",
	)
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	router.POST("/authorize/consent", ts.HttpProvider.ConsentHandler())
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	// The grant is keyed on the EXACT query string, so the consent record and
	// every later /authorize must carry byte-identical parameters.
	qs := url.Values{}
	qs.Set("response_type", "code")
	qs.Set("client_id", thirdParty.ClientID)
	qs.Set("redirect_uri", redirectURI)
	qs.Set("state", "xyz")
	qs.Set("response_mode", "query")
	qs.Set("scope", "openid profile email") // far beyond the agent's ceiling
	qs.Set("code_challenge", s256Challenge("a-verifier-long-enough-to-be-valid-00000000000"))
	qs.Set("code_challenge_method", "S256")
	query := qs.Encode()

	plantConsent := func(t *testing.T) string {
		t.Helper()
		id := uuid.NewString()
		payload, mErr := json.Marshal(map[string]interface{}{
			"client_id":    thirdParty.ClientID,
			"client_name":  "Third Party App",
			"redirect_uri": redirectURI,
			"query":        query,
			"user_id":      user.ID,
			"scopes":       []string{"openid", "profile", "email"},
		})
		require.NoError(t, mErr)
		require.NoError(t, ts.MemoryStoreProvider.SetState("cimd_consent:"+id, string(payload)))
		return id
	}

	approveWithDelegatedToken := func(t *testing.T, consentID string) *httptest.ResponseRecorder {
		t.Helper()
		form := url.Values{"consent_id": {consentID}, "action": {"approve"}}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/authorize/consent", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+delegated)
		// ValidateDelegatedAccessToken derives the expected audience from the
		// request host (parsers.GetHost); httptest defaults it to example.com,
		// so without this the token is refused on AUDIENCE and the
		// authorization check under test is never reached.
		req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
		router.ServeHTTP(w, req)
		return w
	}

	authorizeGET := func(sessionToken string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/authorize?"+query, nil)
		req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
		if sessionToken != "" {
			req.AddCookie(&http.Cookie{Name: constants.AppCookieName + "_session", Value: sessionToken})
		}
		router.ServeHTTP(w, req)
		return w
	}

	// The regression guard. Before the fix this returned 302 and the approval
	// was recorded; an openid-only delegated token could grant a third-party
	// client every scope it asked for, unbounded by the agent's own ceiling.
	t.Run("a delegated token cannot approve consent", func(t *testing.T) {
		w := approveWithDelegatedToken(t, plantConsent(t))
		require.Equal(t, http.StatusForbidden, w.Code,
			"a delegated token must not approve a consent grant: %s", w.Body.String())
		assert.Contains(t, w.Body.String(), "your own session",
			"the refusal must be legible to the person in front of the screen")
		assert.NotContains(t, w.Body.String(), user.ID,
			"the page must not disclose the subject of the consent")
	})

	// The no-breakage proof: the normal browser flow is untouched. The session
	// cookie is checked BEFORE the bearer token in the resolver and the session
	// branch never sets ActorID, so a first-party caller — including an agent
	// driving the user's own browser with their cookie — still approves.
	t.Run("a first-party session still approves and mints a code", func(t *testing.T) {
		consentID := plantConsent(t)
		sessionToken := sessionForExistingUser(t, ts, user)

		form := url.Values{"consent_id": {consentID}, "action": {"approve"}}
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/authorize/consent", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
		req.AddCookie(&http.Cookie{Name: constants.AppCookieName + "_session", Value: sessionToken})
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusFound, w.Code,
			"a first-party session must still approve: %s", w.Body.String())

		got := authorizeGET(sessionToken)
		require.Equal(t, http.StatusFound, got.Code, "body: %s", got.Body.String())
		loc, pErr := url.Parse(got.Header().Get("Location"))
		require.NoError(t, pErr)
		assert.Equal(t, redirectURI, loc.Scheme+"://"+loc.Host+loc.Path,
			"the code goes to the client's registered redirect_uri")
		assert.NotEmpty(t, loc.Query().Get("code"),
			"the normal consent flow still mints an authorization code")
	})

	// FACT 2 — the disputed impact claim. The advisory says this works "with no
	// session cookie at all" and that "the resulting authorization code is
	// delivered to the client's redirect_uri". /authorize reads its session from
	// the cookie only (authorize.go:564) with no bearer fallback, so the
	// redirect cannot be redeemed by the caller that produced it.
	t.Run("the login-handoff code is not redeemable without authentication", func(t *testing.T) {
		w := authorizeGET("")
		loc := w.Header().Get("Location")
		t.Logf("authorize without cookie -> %d %q", w.Code, loc)

		// The caller is sent to the authorizer's OWN login UI, not to the
		// client. Asserting on the absence of "code=" would be wrong: the login
		// handoff legitimately carries a pre-generated code so the UI can finish
		// the flow after the user authenticates (authorize.go:500).
		parsed, pErr := url.Parse(loc)
		require.NoError(t, pErr)
		assert.False(t, strings.HasPrefix(loc, redirectURI),
			"must not redirect to the third party without a session: %q", loc)
		assert.Equal(t, "/app", parsed.Path, "the caller is sent to the login UI")

		// THE decisive check. That handoff code is visible to the cookie-less
		// caller. If it were redeemable the advisory's impact claim would hold,
		// so exchanging it is the experiment, not a formality: the code is not
		// yet bound to an authenticated user (the session component of the
		// stored challenge is appended only after login), so the token endpoint
		// must refuse it.
		handoffCode := parsed.Query().Get("code")
		require.NotEmpty(t, handoffCode, "precondition: the login handoff carries a code")

		f := url.Values{}
		f.Set("grant_type", "authorization_code")
		f.Set("code", handoffCode)
		f.Set("redirect_uri", redirectURI)
		f.Set("client_id", thirdParty.ClientID)
		f.Set("code_verifier", "a-verifier-long-enough-to-be-valid-00000000000")
		tw := httptest.NewRecorder()
		treq := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(f.Encode()))
		treq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		treq.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
		router.ServeHTTP(tw, treq)
		t.Logf("token exchange of the handoff code -> %d body=%.240s", tw.Code, tw.Body.String())

		assert.NotEqual(t, http.StatusOK, tw.Code,
			"the login-handoff code must NOT be redeemable without authentication")
		assert.NotContains(t, tw.Body.String(), "access_token",
			"no access token may be issued from the handoff code")
	})

}

// sessionForExistingUser mints a browser session for an ALREADY-CREATED user.
// mcpSession creates its own user, which is no use when the test needs a session
// for the specific victim whose consent was planted.
func sessionForExistingUser(t *testing.T, ts *testSetup, user *schemas.User) string {
	t.Helper()
	nonce := uuid.NewString()
	sessionData, sessionToken, sessionExpiresAt, err := ts.TokenProvider.CreateSessionToken(&token.AuthTokenConfig{
		User:        user,
		Nonce:       nonce,
		Roles:       ts.Config.DefaultRoles,
		Scope:       []string{"openid", "profile", "email"},
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		constants.AuthRecipeMethodBasicAuth+":"+user.ID,
		constants.TokenTypeSessionToken+"_"+sessionData.Nonce, sessionToken, sessionExpiresAt))
	return sessionToken
}

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

	"github.com/authorizerdev/authorizer/internal/codestate"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const magicLinkCallback = "http://localhost:3000/callback"

func magicLinkTestConfig() *config.Config {
	cfg := getTestConfig()
	cfg.EnableMagicLinkLogin = true
	cfg.IsEmailServiceEnabled = true
	cfg.EnableEmailVerification = true
	return cfg
}

// magicLinkRouter mounts the three endpoints a magic link started from
// /authorize travels through.
func magicLinkRouter(ts *testSetup) *gin.Engine {
	r := gin.New()
	r.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	r.GET("/verify_email", ts.HttpProvider.VerifyEmailHandler())
	r.POST("/oauth/token", ts.HttpProvider.TokenHandler())
	return r
}

func serveGET(ts *testSetup, r *gin.Engine, target string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
	for _, c := range cookies {
		req.AddCookie(c)
	}
	r.ServeHTTP(w, req)
	return w
}

// clickLink follows the emailed link as a browser would, against the router,
// presenting the given cookies. A browser percent-encodes a literal space
// before sending the request, so that is the only normalisation applied.
func clickLink(t *testing.T, ts *testSetup, r *gin.Engine, link string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	u, err := url.Parse(link)
	require.NoError(t, err)
	return serveGET(ts, r, strings.ReplaceAll(u.RequestURI(), " ", "%20"), cookies...)
}

func redeemCode(ts *testSetup, r *gin.Engine, code, verifier, redirectURI string) (int, map[string]interface{}) {
	f := url.Values{}
	f.Set("grant_type", "authorization_code")
	f.Set("code", code)
	f.Set("client_id", ts.Config.ClientID)
	f.Set("redirect_uri", redirectURI)
	if verifier != "" {
		f.Set("code_verifier", verifier)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(f.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Authorizer-URL", testAuthorizerHost(ts))
	r.ServeHTTP(w, req)
	body := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

// authorizeFlow is what one browser holds after an unauthenticated PKCE
// /authorize: the state, the pre-generated code the login UI URL carries, and
// whatever cookies /authorize set on that browser.
type authorizeFlow struct {
	State       string
	LoginUICode string
	Cookies     []*http.Cookie
}

func startAuthorize(t *testing.T, ts *testSetup, r *gin.Engine, verifier string) authorizeFlow {
	t.Helper()
	state := uuid.NewString()
	qs := url.Values{}
	qs.Set("response_type", "code")
	qs.Set("response_mode", "query")
	qs.Set("client_id", ts.Config.ClientID)
	qs.Set("redirect_uri", magicLinkCallback)
	qs.Set("state", state)
	qs.Set("scope", "openid profile email")
	qs.Set("code_challenge", s256Challenge(verifier))
	qs.Set("code_challenge_method", "S256")
	aw := serveGET(ts, r, "/authorize?"+qs.Encode())
	require.Equal(t, http.StatusFound, aw.Code, "authorize: %s", aw.Body.String())
	loginUI, err := url.Parse(aw.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "/app", loginUI.Path, "an unauthenticated /authorize detours through the login UI")
	require.Equal(t, state, loginUI.Query().Get("state"))
	return authorizeFlow{State: state, LoginUICode: loginUI.Query().Get("code"), Cookies: aw.Result().Cookies()}
}

// sendMagicLink calls magic_link_login for email with the given state and
// redirect, and returns the link from the email actually sent.
func sendMagicLink(t *testing.T, ts *testSetup, email, state, redirect string) string {
	t.Helper()
	_, ctx := createContext(ts)
	_, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
		Email:       email,
		State:       refs.NewStringRef(state),
		Scope:       []string{"openid", "profile", "email"},
		RedirectURI: refs.NewStringRef(redirect),
	})
	require.NoError(t, err)
	return ts.SentEmails.waitForVerificationURL(t, email, constants.VerificationTypeMagicLinkLogin)
}

// authorizeAndClickMagicLink runs /authorize → magic_link_login → clicks the
// emailed link in the SAME browser that ran /authorize. It returns the state
// and the raw /verify_email response.
func authorizeAndClickMagicLink(t *testing.T, ts *testSetup, r *gin.Engine, verifier, magicRedirect string) (string, *httptest.ResponseRecorder) {
	t.Helper()
	flow := startAuthorize(t, ts, r, verifier)
	link := sendMagicLink(t, ts, "magic_authz_"+uuid.NewString()+"@authorizer.dev", flow.State, magicRedirect)
	return flow.State, clickLink(t, ts, r, link, flow.Cookies...)
}

// startAuthorizeMagicLink is authorizeAndClickMagicLink for the happy path,
// returning the parsed /verify_email redirect.
func startAuthorizeMagicLink(t *testing.T, ts *testSetup, r *gin.Engine, verifier string) *url.URL {
	t.Helper()
	state, vw := authorizeAndClickMagicLink(t, ts, r, verifier, magicLinkCallback)
	require.Equal(t, http.StatusTemporaryRedirect, vw.Code, "verify_email: %s", vw.Body.String())
	loc, err := url.Parse(vw.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, magicLinkCallback, loc.Scheme+"://"+loc.Host+loc.Path)
	require.Equal(t, state, loc.Query().Get("state"))
	return loc
}

// TestMagicLinkAuthorizeCodeIsRedeemable pins that a magic link started from a
// PKCE /authorize request yields a `code` that /oauth/token accepts — exactly
// like a code minted after password login — and that it is bound to the PKCE
// verifier and single-use.
//
// Previously /verify_email put the pre-generated code in the redirect but never
// stored the code state (the SetState was commented out), so every redemption
// failed with invalid_grant.
func TestMagicLinkAuthorizeCodeIsRedeemable(t *testing.T) {
	ts := initTestSetup(t, magicLinkTestConfig())
	r := magicLinkRouter(ts)
	const verifier = "magic-link-verifier-long-enough-to-be-valid-0000000"

	t.Run("correct verifier redeems once", func(t *testing.T) {
		loc := startAuthorizeMagicLink(t, ts, r, verifier)
		code := loc.Query().Get("code")
		require.NotEmpty(t, code, "an authorize-context magic link must deliver a code")

		status, body := redeemCode(ts, r, code, verifier, magicLinkCallback)
		require.Equal(t, http.StatusOK, status, "token exchange: %v", body)
		assert.NotEmpty(t, body["access_token"])
		assert.NotEmpty(t, body["id_token"])

		status, body = redeemCode(ts, r, code, verifier, magicLinkCallback)
		assert.Equal(t, http.StatusBadRequest, status, "a code must be single-use: %v", body)
		assert.Equal(t, "invalid_grant", body["error"])
	})

	t.Run("wrong verifier is refused", func(t *testing.T) {
		loc := startAuthorizeMagicLink(t, ts, r, verifier)
		code := loc.Query().Get("code")
		require.NotEmpty(t, code)

		status, body := redeemCode(ts, r, code, "an-entirely-different-verifier-000000000000000000", magicLinkCallback)
		assert.Equal(t, http.StatusBadRequest, status, "%v", body)
		assert.Equal(t, "invalid_grant", body["error"])
		assert.Empty(t, body["access_token"])
	})

	t.Run("missing verifier is refused", func(t *testing.T) {
		loc := startAuthorizeMagicLink(t, ts, r, verifier)
		code := loc.Query().Get("code")
		require.NotEmpty(t, code)

		status, body := redeemCode(ts, r, code, "", magicLinkCallback)
		assert.Equal(t, http.StatusBadRequest, status, "%v", body)
		assert.Empty(t, body["access_token"])
	})
}

// TestMagicLinkWithoutAuthorizeContextIssuesNoCode pins that /verify_email
// does not invent a code when the magic link carries no /authorize state.
func TestMagicLinkWithoutAuthorizeContextIssuesNoCode(t *testing.T) {
	ts := initTestSetup(t, magicLinkTestConfig())
	r := magicLinkRouter(ts)
	_, ctx := createContext(ts)

	email := "magic_noauthz_" + uuid.NewString() + "@authorizer.dev"
	_, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
		Email:       email,
		State:       refs.NewStringRef(uuid.NewString()), // never stored by /authorize
		RedirectURI: refs.NewStringRef(magicLinkCallback),
	})
	require.NoError(t, err)

	vw := clickLink(t, ts, r, ts.SentEmails.waitForVerificationURL(t, email, constants.VerificationTypeMagicLinkLogin))
	require.Equal(t, http.StatusTemporaryRedirect, vw.Code, vw.Body.String())
	loc, err := url.Parse(vw.Header().Get("Location"))
	require.NoError(t, err)
	assert.Empty(t, loc.Query().Get("code"))
}

// TestMagicLinkWithExactRedirectURIs pins that a magic link works when the
// deployment declares --redirect-uris. The link used to concatenate
// redirect_uri raw, and magic_link_login appended ?roles=&state=&scope= to the
// redirect itself, so /verify_email saw "<uri>?roles=..." — no longer an exact
// match — and answered 400 "invalid redirect uri".
func TestMagicLinkWithExactRedirectURIs(t *testing.T) {
	cases := []struct {
		name     string
		redirect string
	}{
		{"plain URI", magicLinkCallback},
		{"URI with its own query string", magicLinkCallback + "?tenant=acme&x=a%26b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := magicLinkTestConfig()
			cfg.RedirectURIs = []string{tc.redirect}
			ts := initTestSetup(t, cfg)
			r := magicLinkRouter(ts)
			_, ctx := createContext(ts)

			email := "magic_exact_" + uuid.NewString() + "@authorizer.dev"
			state := uuid.NewString()
			_, err := ts.GraphQLProvider.MagicLinkLogin(ctx, &model.MagicLinkLoginRequest{
				Email:       email,
				State:       refs.NewStringRef(state),
				Scope:       []string{"openid", "email"},
				RedirectURI: refs.NewStringRef(tc.redirect),
			})
			require.NoError(t, err)

			link := ts.SentEmails.waitForVerificationURL(t, email, constants.VerificationTypeMagicLinkLogin)
			lu, err := url.Parse(link)
			require.NoError(t, err)
			assert.Equal(t, tc.redirect, lu.Query().Get("redirect_uri"),
				"redirect_uri must survive the link's own query parsing intact")

			vw := clickLink(t, ts, r, link)
			require.Equal(t, http.StatusTemporaryRedirect, vw.Code, "verify_email: %s", vw.Body.String())

			loc, err := url.Parse(vw.Header().Get("Location"))
			require.NoError(t, err)
			want, err := url.Parse(tc.redirect)
			require.NoError(t, err)
			assert.Equal(t, want.Scheme+"://"+want.Host+want.Path, loc.Scheme+"://"+loc.Host+loc.Path)
			for k, v := range want.Query() {
				assert.Equal(t, v, loc.Query()[k], "the redirect's own %q param must survive", k)
			}
			assert.Equal(t, state, loc.Query().Get("state"))
			assert.Empty(t, loc.Query().Get("roles"), "roles are not smuggled through the redirect URI")
		})
	}
}

// TestMagicLinkAuthorizeCodeWithheldByMFA pins that no code is issued when the
// MFA gate withholds the session: the code state would otherwise bind a code to
// a session the user has not finished establishing. The authorize state must
// survive so the flow can resume after the second factor.
func TestMagicLinkAuthorizeCodeWithheldByMFA(t *testing.T) {
	cfg := magicLinkTestConfig()
	cfg.EnableMFA = true
	cfg.EnableTOTPLogin = true
	cfg.EnforceMFA = true
	ts := initTestSetup(t, cfg)
	r := magicLinkRouter(ts)

	state, vw := authorizeAndClickMagicLink(t, ts, r, "magic-link-verifier-long-enough-to-be-valid-0000000", magicLinkCallback)
	require.Equal(t, http.StatusTemporaryRedirect, vw.Code, vw.Body.String())
	loc, err := url.Parse(vw.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "1", loc.Query().Get("mfa_required"), "precondition: the MFA gate withheld the session")
	assert.Empty(t, loc.Query().Get("code"), "no code while MFA withholds the session")
	assert.NotContains(t, loc.String(), "access_token")

	remaining, _ := ts.MemoryStoreProvider.GetState(state)
	assert.NotEmpty(t, remaining, "the authorize state must not be consumed by a withheld session")
}

// TestMagicLinkAuthorizeCodeOnlyToAuthorizeRedirect pins that the code goes
// only to the redirect_uri the /authorize request recorded (RFC 6749 §4.1.2).
// A magic link pointed at a different, merely allowed, URI still signs the
// user in (cookie) but carries no code.
func TestMagicLinkAuthorizeCodeOnlyToAuthorizeRedirect(t *testing.T) {
	ts := initTestSetup(t, magicLinkTestConfig())
	r := magicLinkRouter(ts)

	_, vw := authorizeAndClickMagicLink(t, ts, r, "magic-link-verifier-long-enough-to-be-valid-0000000", "http://localhost:3000/elsewhere")
	require.Equal(t, http.StatusTemporaryRedirect, vw.Code, vw.Body.String())
	loc, err := url.Parse(vw.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "/elsewhere", loc.Path)
	assert.Empty(t, loc.Query().Get("code"))
}

// TestMagicLinkAuthorizeCodeRequiresInitiatingBrowser pins that a magic link
// only completes an /authorize request in the browser that started it.
//
// magic_link_login is unauthenticated and accepts any `state`, so the state
// alone says nothing about who clicks the link, while whoever ran /authorize
// holds the PKCE verifier. /authorize therefore binds the request to the
// initiating browser with a cookie, and /verify_email issues a code only when
// that binding is presented. Without it the click still signs the user in
// (session cookie) but carries no code.
func TestMagicLinkAuthorizeCodeRequiresInitiatingBrowser(t *testing.T) {
	ts := initTestSetup(t, magicLinkTestConfig())
	r := magicLinkRouter(ts)
	const verifier = "initiator-verifier-long-enough-to-be-valid-00000000"

	assertNoCodeAndNothingRedeemable := func(t *testing.T, flow authorizeFlow, vw *httptest.ResponseRecorder) {
		t.Helper()
		require.Equal(t, http.StatusTemporaryRedirect, vw.Code, vw.Body.String())
		loc, err := url.Parse(vw.Header().Get("Location"))
		require.NoError(t, err)
		redirected := loc.Query().Get("code")
		assert.Empty(t, redirected, "no code without the initiating browser's binding")
		for _, c := range []string{redirected, flow.LoginUICode} {
			if c == "" {
				continue
			}
			status, body := redeemCode(ts, r, c, verifier, magicLinkCallback)
			assert.NotEqual(t, http.StatusOK, status, "the /authorize initiator must not obtain tokens: %v", body)
			assert.Empty(t, body["access_token"])
		}
	}

	t.Run("link clicked without the binding cookie issues no code", func(t *testing.T) {
		flow := startAuthorize(t, ts, r, verifier)
		link := sendMagicLink(t, ts, "magic_bind_"+uuid.NewString()+"@authorizer.dev", flow.State, magicLinkCallback)
		assertNoCodeAndNothingRedeemable(t, flow, clickLink(t, ts, r, link))
	})

	t.Run("a binding from a different /authorize request does not match", func(t *testing.T) {
		mine := startAuthorize(t, ts, r, verifier)
		theirs := startAuthorize(t, ts, r, verifier)
		link := sendMagicLink(t, ts, "magic_bind_"+uuid.NewString()+"@authorizer.dev", theirs.State, magicLinkCallback)
		assertNoCodeAndNothingRedeemable(t, theirs, clickLink(t, ts, r, link, mine.Cookies...))
	})

	t.Run("the initiating browser gets a code redeemable once", func(t *testing.T) {
		loc := startAuthorizeMagicLink(t, ts, r, verifier)
		code := loc.Query().Get("code")
		require.NotEmpty(t, code)
		status, body := redeemCode(ts, r, code, verifier, magicLinkCallback)
		require.Equal(t, http.StatusOK, status, "%v", body)
		status, _ = redeemCode(ts, r, code, verifier, magicLinkCallback)
		assert.Equal(t, http.StatusBadRequest, status)
	})
}

// TestMagicLinkRedirectValidatedAgainstAuthorizeClient pins that /verify_email
// validates the redirect against the client of the /authorize request the link
// completes, not always the deployment's own client. With --redirect-uris set,
// a third-party client's registered redirect — already accepted by /authorize
// — used to be refused here with 400 because it is not in the deployment's own
// list.
func TestMagicLinkRedirectValidatedAgainstAuthorizeClient(t *testing.T) {
	cfg := magicLinkTestConfig()
	cfg.RedirectURIs = []string{magicLinkCallback}
	ts := initTestSetup(t, cfg)
	r := magicLinkRouter(ts)

	const thirdPartyRedirect = "http://localhost:3000/third-party-cb"
	client, err := ts.StorageProvider.AddClient(context.Background(), &schemas.Client{
		Name:          "third-party-" + uuid.NewString(),
		Kind:          constants.ClientKindInteractive,
		RedirectURIs:  thirdPartyRedirect,
		AllowedScopes: "openid,profile,email",
		GrantTypes:    "authorization_code,refresh_token",
		IsActive:      true,
	})
	require.NoError(t, err)

	const binding = "third-party-browser-binding"
	state := uuid.NewString()
	require.NoError(t, ts.MemoryStoreProvider.SetState(state, codestate.EncodeAuthorize(codestate.Authorize{
		Code:           uuid.NewString(),
		Challenge:      s256Challenge("third-party-verifier-long-enough-to-be-valid-000") + "::S256",
		Nonce:          uuid.NewString(),
		RedirectURI:    thirdPartyRedirect,
		ClientID:       client.ClientID,
		BrowserBinding: crypto.HashSessionValue(binding),
	})))

	link := sendMagicLink(t, ts, "magic_3p_"+uuid.NewString()+"@authorizer.dev", state, thirdPartyRedirect)
	vw := clickLink(t, ts, r, link, &http.Cookie{Name: constants.AuthorizeBindingCookieName, Value: binding})
	require.Equal(t, http.StatusTemporaryRedirect, vw.Code, "verify_email: %s", vw.Body.String())
	loc, err := url.Parse(vw.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, thirdPartyRedirect, loc.Scheme+"://"+loc.Host+loc.Path)
	assert.NotEmpty(t, loc.Query().Get("code"))
}

// TestMagicLinkThirdPartyRedirectNeedsBrowserBinding pins that the link's
// `state` can only widen redirect validation to another client's registered
// URIs when the clicking browser is the one that ran that /authorize request.
// The state is caller-supplied (magic_link_login is unauthenticated), so
// honouring its client_id unconditionally would let anyone point a genuine
// verification email at a URI registered by a client they control, bypassing
// the exact --redirect-uris match for the deployment's own client.
func TestMagicLinkThirdPartyRedirectNeedsBrowserBinding(t *testing.T) {
	cfg := magicLinkTestConfig()
	cfg.RedirectURIs = []string{magicLinkCallback}
	ts := initTestSetup(t, cfg)
	r := magicLinkRouter(ts)

	const otherRedirect = "http://localhost:3000/other-client-cb"
	client, err := ts.StorageProvider.AddClient(context.Background(), &schemas.Client{
		Name:          "other-" + uuid.NewString(),
		Kind:          constants.ClientKindInteractive,
		RedirectURIs:  otherRedirect,
		AllowedScopes: "openid,profile,email",
		GrantTypes:    "authorization_code,refresh_token",
		IsActive:      true,
	})
	require.NoError(t, err)

	// Each case gets its own state and link: clicking consumes the token.
	freshLink := func(t *testing.T) string {
		t.Helper()
		state := uuid.NewString()
		require.NoError(t, ts.MemoryStoreProvider.SetState(state, codestate.EncodeAuthorize(codestate.Authorize{
			Code:           uuid.NewString(),
			Challenge:      s256Challenge("other-verifier-long-enough-to-be-valid-0000000000") + "::S256",
			Nonce:          uuid.NewString(),
			RedirectURI:    otherRedirect,
			ClientID:       client.ClientID,
			BrowserBinding: crypto.HashSessionValue("the-initiators-binding"),
		})))
		return sendMagicLink(t, ts, "magic_or_"+uuid.NewString()+"@authorizer.dev", state, otherRedirect)
	}

	t.Run("without the binding cookie the redirect is refused", func(t *testing.T) {
		vw := clickLink(t, ts, r, freshLink(t))
		assert.Equal(t, http.StatusBadRequest, vw.Code, "must not redirect to another client's URI: %s", vw.Header().Get("Location"))
		assert.Contains(t, vw.Body.String(), "invalid redirect uri")
		assert.Empty(t, vw.Header().Get("Location"))
	})

	t.Run("a non-matching binding cookie is refused too", func(t *testing.T) {
		vw := clickLink(t, ts, r, freshLink(t), &http.Cookie{Name: constants.AuthorizeBindingCookieName, Value: "someone-elses"})
		assert.Equal(t, http.StatusBadRequest, vw.Code, "must not redirect to another client's URI: %s", vw.Header().Get("Location"))
		assert.Contains(t, vw.Body.String(), "invalid redirect uri")
	})

	t.Run("the initiating browser is redirected", func(t *testing.T) {
		vw := clickLink(t, ts, r, freshLink(t), &http.Cookie{Name: constants.AuthorizeBindingCookieName, Value: "the-initiators-binding"})
		require.Equal(t, http.StatusTemporaryRedirect, vw.Code, vw.Body.String())
		assert.True(t, strings.HasPrefix(vw.Header().Get("Location"), otherRedirect+"?"))
	})
}

// TestAuthorizeBindingCookieOnlyOnLoginDetour pins that /authorize sets the
// browser-binding cookie only when it sends the browser to the login UI. A
// request answered from an existing session, or a prompt=none renew, would
// otherwise overwrite the binding of a magic-link flow pending in another tab.
func TestAuthorizeBindingCookieOnlyOnLoginDetour(t *testing.T) {
	ts := initTestSetup(t, magicLinkTestConfig())
	r := magicLinkRouter(ts)
	_, ctx := createContext(ts)
	const verifier = "binding-detour-verifier-long-enough-000000000"

	hasBindingCookie := func(cookies []*http.Cookie) bool {
		for _, c := range cookies {
			if c.Name == constants.AuthorizeBindingCookieName {
				return true
			}
		}
		return false
	}
	authorizeQuery := func(extra map[string]string) string {
		qs := url.Values{}
		qs.Set("response_type", "code")
		qs.Set("response_mode", "query")
		qs.Set("client_id", ts.Config.ClientID)
		qs.Set("redirect_uri", magicLinkCallback)
		qs.Set("state", uuid.NewString())
		qs.Set("scope", "openid profile email")
		qs.Set("code_challenge", s256Challenge(verifier))
		qs.Set("code_challenge_method", "S256")
		for k, v := range extra {
			qs.Set(k, v)
		}
		return "/authorize?" + qs.Encode()
	}

	t.Run("login detour sets it", func(t *testing.T) {
		assert.True(t, hasBindingCookie(startAuthorize(t, ts, r, verifier).Cookies))
	})

	t.Run("prompt=none without a session does not", func(t *testing.T) {
		w := serveGET(ts, r, authorizeQuery(map[string]string{"prompt": "none"}))
		require.Equal(t, http.StatusFound, w.Code, w.Body.String())
		assert.Contains(t, w.Header().Get("Location"), "login_required")
		assert.False(t, hasBindingCookie(w.Result().Cookies()))
	})

	t.Run("an existing session does not", func(t *testing.T) {
		email := "binding_session_" + uuid.NewString() + "@authorizer.dev"
		const password = "Password@123"
		_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{Email: &email, Password: password, ConfirmPassword: password})
		require.NoError(t, err)
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		sessionToken := sessionForExistingUser(t, ts, user)

		w := serveGET(ts, r, authorizeQuery(nil), &http.Cookie{Name: constants.AppCookieName + "_session", Value: sessionToken})
		require.Equal(t, http.StatusFound, w.Code, w.Body.String())
		loc, err := url.Parse(w.Header().Get("Location"))
		require.NoError(t, err)
		require.NotEmpty(t, loc.Query().Get("code"), "precondition: answered from the session, got %s", loc)
		assert.False(t, hasBindingCookie(w.Result().Cookies()))
	})
}

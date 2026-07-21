package integration_tests

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
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
	"github.com/authorizerdev/authorizer/internal/token"
)

// refreshFamilyKeyPrefix mirrors the (unexported) constant in
// internal/http_handlers/token.go. The reuse-detection family record for a
// lineage is stored under "<loginMethod>:<userID>" at this sub-key +
// <familyID>. Tests reach into it to simulate elapsed time for the grace
// window without sleeping.
const refreshFamilyKeyPrefix = "refresh_family_"

// offlineAccessSessionForEmail logs an ALREADY-signed-up user in for a fresh
// offline_access session and returns a router plus an authorization code +
// PKCE verifier ready to exchange. Calling it twice for the same email yields
// two independent refresh-token lineages (families) under the same session key
// — the setup needed to prove reuse revocation is scoped to one family.
func offlineAccessSessionForEmail(t *testing.T, ts *testSetup, clientID, email string) (http.Handler, string, string) {
	t.Helper()
	_, ctx := createContext(ts)

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	nonce := uuid.New().String()
	scope := []string{"openid", "profile", "email", "offline_access"}
	sessionData, sessionToken, sessionExpiresAt, err := ts.TokenProvider.CreateSessionToken(&token.AuthTokenConfig{
		User:        user,
		Nonce:       nonce,
		Roles:       ts.Config.DefaultRoles,
		Scope:       scope,
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
	})
	require.NoError(t, err)

	sessionKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		sessionKey, constants.TokenTypeSessionToken+"_"+sessionData.Nonce, sessionToken, sessionExpiresAt))

	verifier := uuid.New().String() + uuid.New().String()
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	router := gin.New()
	router.GET("/authorize", ts.HttpProvider.AuthorizeHandler())
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	qs := url.Values{}
	qs.Set("client_id", clientID)
	qs.Set("response_type", "code")
	qs.Set("redirect_uri", "http://localhost:3000/callback")
	qs.Set("code_challenge", challenge)
	qs.Set("code_challenge_method", "S256")
	qs.Set("state", uuid.New().String())
	qs.Set("response_mode", "query")
	qs.Set("scope", strings.Join(scope, " "))

	code := codeFromRedirect(t, doAuthorizeGET(router, qs, sessionToken))
	return router, code, verifier
}

func signupUser(t *testing.T, ts *testSetup) string {
	t.Helper()
	_, ctx := createContext(ts)
	email := "hardening_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	})
	require.NoError(t, err)
	return email
}

func exchangeForRefresh(t *testing.T, router http.Handler, code, verifier string, basic []string) string {
	t.Helper()
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	form.Set("redirect_uri", "http://localhost:3000/callback")
	w := exchangeCode(router, form, basic)
	require.Equal(t, http.StatusOK, w.Code, "code exchange body: %s", w.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	rt, _ := body["refresh_token"].(string)
	require.NotEmpty(t, rt, "offline_access must yield a refresh_token")
	return rt
}

func doRefresh(router http.Handler, rt string, basic []string) *httptest.ResponseRecorder {
	f := url.Values{}
	f.Set("grant_type", "refresh_token")
	f.Set("refresh_token", rt)
	return exchangeCode(router, f, basic)
}

func refreshOK(t *testing.T, router http.Handler, rt string, basic []string) string {
	t.Helper()
	w := doRefresh(router, rt, basic)
	require.Equal(t, http.StatusOK, w.Code, "refresh body: %s", w.Body.String())
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	rt2, _ := body["refresh_token"].(string)
	require.NotEmpty(t, rt2)
	return rt2
}

// Item 1 (the HIGH fix): refresh-token reuse revocation MUST be scoped to the
// compromised token's own family — replaying one retired token must NOT log the
// user out of their OTHER live sessions. Before the fix, the token endpoint
// called DeleteAllUserSessions(userID), so any single leaked/retired refresh
// token was an unauthenticated forced-logout DoS across every session of that
// user. This test builds TWO independent refresh lineages for the SAME user
// (same session key), triggers genuine reuse on one, and asserts the other
// survives.
func TestOAuth21_RefreshTokenReuse_ScopedToFamily(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	registerTestClient(t, ts, "family-client", "family-secret")
	basic := []string{"family-client", "family-secret"}

	email := signupUser(t, ts)

	// Family A and family B: two separate logins of the same user.
	routerA, codeA, verifierA := offlineAccessSessionForEmail(t, ts, "family-client", email)
	familyARefresh := exchangeForRefresh(t, routerA, codeA, verifierA, basic)

	routerB, codeB, verifierB := offlineAccessSessionForEmail(t, ts, "family-client", email)
	familyBRefresh := exchangeForRefresh(t, routerB, codeB, verifierB, basic)

	// Advance family A twice so its original token is genuinely stale (not the
	// immediate predecessor covered by the grace window), then replay it.
	aRotated := refreshOK(t, routerA, familyARefresh, basic)
	aLive := refreshOK(t, routerA, aRotated, basic)

	reuse := doRefresh(routerA, familyARefresh, basic)
	assert.Equal(t, http.StatusBadRequest, reuse.Code, "reuse body: %s", reuse.Body.String())

	// Family A's live token is revoked (breach response, scoped to family A).
	deadA := doRefresh(routerA, aLive, basic)
	assert.Equal(t, http.StatusBadRequest, deadA.Code,
		"family A live token must be revoked after reuse in family A: %s", deadA.Body.String())

	// The DoS fix: family B — an unrelated session of the SAME user — is
	// untouched and still refreshes. Under the old whole-user revocation this
	// would have been wiped too.
	survivor := doRefresh(routerB, familyBRefresh, basic)
	assert.Equal(t, http.StatusOK, survivor.Code,
		"an unrelated session of the same user must survive reuse in another family: %s", survivor.Body.String())
}

// Item 1 grace window: a benign double-submit (multi-tab SPA / network retry)
// replays the just-rotated-away token against its own still-live successor.
// Within the short grace window that is treated as an ordinary race — the
// replay fails invalid_grant but the live session is NOT revoked. After the
// window the SAME replay is treated as genuine reuse and revokes the family.
func TestOAuth21_RefreshTokenReuse_GraceWindow(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	registerTestClient(t, ts, "grace-client", "grace-secret")
	basic := []string{"grace-client", "grace-secret"}

	t.Run("within window: benign double-submit does not revoke", func(t *testing.T) {
		email := signupUser(t, ts)
		router, code, verifier := offlineAccessSessionForEmail(t, ts, "grace-client", email)
		original := exchangeForRefresh(t, router, code, verifier, basic)

		live := refreshOK(t, router, original, basic) // rotate once: original -> live

		// Immediately replay the just-rotated-away token (the immediate
		// predecessor) — a double-submit race.
		replay := doRefresh(router, original, basic)
		assert.Equal(t, http.StatusBadRequest, replay.Code, "the replayed token itself is invalid: %s", replay.Body.String())

		// The live session must NOT have been revoked — it still refreshes.
		stillAlive := doRefresh(router, live, basic)
		assert.Equal(t, http.StatusOK, stillAlive.Code,
			"a benign double-submit within the grace window must not revoke the live session: %s", stillAlive.Body.String())
	})

	t.Run("after window: same replay is genuine reuse and revokes", func(t *testing.T) {
		email := signupUser(t, ts)
		router, code, verifier := offlineAccessSessionForEmail(t, ts, "grace-client", email)
		original := exchangeForRefresh(t, router, code, verifier, basic)

		live := refreshOK(t, router, original, basic) // rotate once: original -> live

		// Simulate the grace window having elapsed by ageing the family record's
		// rotated_at far into the past (no sleep). familyID + owner come from the
		// live refresh token's claims.
		ageFamilyRecordPast(t, ts, live)

		replay := doRefresh(router, original, basic)
		assert.Equal(t, http.StatusBadRequest, replay.Code, "reuse body: %s", replay.Body.String())

		// Now the live session IS revoked — reuse after the window is a breach.
		revoked := doRefresh(router, live, basic)
		assert.Equal(t, http.StatusBadRequest, revoked.Code,
			"reuse of the immediate predecessor after the grace window must revoke the family: %s", revoked.Body.String())
	})
}

// ageFamilyRecordPast rewrites the reuse-detection family record for the given
// live refresh token so its rotated_at is far in the past, simulating an
// elapsed grace window without sleeping.
func ageFamilyRecordPast(t *testing.T, ts *testSetup, liveRefresh string) {
	t.Helper()
	claims := decodeJWTPayloadUnverified(t, liveRefresh)
	familyID, _ := claims["family_id"].(string)
	require.NotEmpty(t, familyID, "live refresh token must carry a family_id")
	sub, _ := claims["sub"].(string)
	loginMethod, _ := claims["login_method"].(string)
	sessionKey := sub
	if loginMethod != "" {
		sessionKey = loginMethod + ":" + sub
	}

	raw, err := ts.MemoryStoreProvider.GetUserSession(sessionKey, refreshFamilyKeyPrefix+familyID)
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	var rec map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(raw), &rec))
	rec["rotated_at"] = int64(1) // far in the past
	out, err := json.Marshal(rec)
	require.NoError(t, err)
	// Keep it live in the store (far-future expiry) so the reuse path finds it.
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		sessionKey, refreshFamilyKeyPrefix+familyID, string(out), 1<<62))
}

// A transient failure looking up the CURRENT live refresh token's own session
// record (a store timeout/blip — GetUserSession cannot distinguish that from
// "key genuinely absent") must NOT be treated as reuse. Simulated here by
// deleting the live token's session entry directly (without a second
// rotation), so the presented nonce still equals the family record's
// LiveNonce — the exact signal handleRefreshTokenReuse uses to recognize
// "this is not a rotated-away token, something else went wrong" and skip
// revocation.
//
// One real rotation happens first and is load-bearing for the test, not
// incidental: a family record only exists once a refresh has actually
// happened (the initial authorization_code exchange writes none), so the
// LiveNonce-guard path can't be reached at all without it.
func TestOAuth21_RefreshTokenReuse_TransientLookupFailureNotRevoked(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	registerTestClient(t, ts, "transient-client", "transient-secret")
	basic := []string{"transient-client", "transient-secret"}

	email := signupUser(t, ts)
	router, code, verifier := offlineAccessSessionForEmail(t, ts, "transient-client", email)
	original := exchangeForRefresh(t, router, code, verifier, basic)
	live := refreshOK(t, router, original, basic) // rotate once: establishes the family record

	claims := decodeJWTPayloadUnverified(t, live)
	nonce, _ := claims["nonce"].(string)
	sub, _ := claims["sub"].(string)
	loginMethod, _ := claims["login_method"].(string)
	require.NotEmpty(t, nonce)
	sessionKey := sub
	if loginMethod != "" {
		sessionKey = loginMethod + ":" + sub
	}

	// Simulate the transient blip: ONLY the live token's own refresh_token_
	// session entry vanishes out from under it (a store timeout mid-lookup) —
	// nothing rotated, so this is still the family's LiveNonce. Overwrite just
	// that one sub-key's expiry to the past (the db-backed store's
	// GetUserSession treats an expired row as "not found" and deletes it on
	// read) rather than DeleteUserSession, which cascades to session_token_/
	// access_token_/refresh_token_ TOGETHER for the nonce and would erase the
	// very sibling entry this test needs intact to prove the point below.
	refreshTokenKey := "refresh_token_" + nonce
	accessTokenKey := "access_token_" + nonce
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(sessionKey, refreshTokenKey, live, 1))
	preAccess, err := ts.MemoryStoreProvider.GetUserSession(sessionKey, accessTokenKey)
	require.NoError(t, err)
	require.NotEmpty(t, preAccess, "test setup: the access token entry must still be present before the replay")

	// Presenting the (now-orphaned-but-still-live-per-the-family-record) token
	// fails validation as an ordinary invalid_grant...
	replay := doRefresh(router, live, basic)
	assert.Equal(t, http.StatusBadRequest, replay.Code, "replay body: %s", replay.Body.String())

	// ...but must NOT have been treated as a breach. If handleRefreshTokenReuse
	// incorrectly ran the genuine-reuse branch, DeleteUserSession(sessionKey,
	// rec.LiveNonce) would have wiped this SAME nonce's access token too — even
	// though nothing about it actually rotated or was stolen.
	postAccess, err := ts.MemoryStoreProvider.GetUserSession(sessionKey, accessTokenKey)
	require.NoError(t, err)
	assert.NotEmpty(t, postAccess,
		"a transient lookup failure on the live token must not revoke the live session (access token entry was wiped)")
}

// Item 2 (MEDIUM): --oauth2-1-strict must reject EVERY response_type that
// delivers a bearer access token into the URL fragment, including the
// front-channel-token hybrids "code token" and "code id_token token" — not
// only "token" / "id_token token". The check is component-exact so pure
// "id_token" (which contains "token" only as a substring) is unaffected.
func TestOAuth21_StrictMode_RejectsFrontChannelTokenHybrids(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	registerTestClient(t, ts, "hybrid-client", "hybrid-secret")

	authorize := func(responseType string) *httptest.ResponseRecorder {
		router, cookie := mcpSession(t, ts, []string{"openid", "profile", "email"})
		qs := url.Values{}
		qs.Set("client_id", "hybrid-client")
		qs.Set("response_type", responseType)
		qs.Set("redirect_uri", "http://localhost:3000/callback")
		qs.Set("state", uuid.New().String())
		qs.Set("response_mode", "fragment")
		qs.Set("nonce", uuid.New().String()) // required whenever id_token is present
		qs.Set("scope", "openid profile email")
		return doAuthorizeGET(router, qs, cookie)
	}

	rejected := []string{"code token", "code id_token token", "token", "id_token token"}
	for _, rt := range rejected {
		t.Run("strict rejects "+rt, func(t *testing.T) {
			ts.Config.OAuth21Strict = true
			w := authorize(rt)
			require.Equal(t, http.StatusFound, w.Code, "body: %s", w.Body.String())
			loc := w.Header().Get("Location")
			assert.Contains(t, loc, "unsupported_response_type", "loc: %s", loc)
			assert.NotContains(t, loc, "access_token=", "no access token may leak into the fragment: %s", loc)
		})
	}

	// Pure id_token (contains "token" only as a substring) is NOT a
	// front-channel access-token flow — strict mode must still allow it.
	t.Run("strict allows pure id_token", func(t *testing.T) {
		ts.Config.OAuth21Strict = true
		w := authorize("id_token")
		require.Equal(t, http.StatusFound, w.Code, "body: %s", w.Body.String())
		loc := w.Header().Get("Location")
		assert.NotContains(t, loc, "unsupported_response_type", "pure id_token must not be rejected by strict mode: %s", loc)
	})

	ts.Config.OAuth21Strict = false
}

// Item 3 (LOW): RFC 8707 §2 requires the resource indicator to be an absolute
// URI with no fragment. An invalid resource is rejected at /authorize with the
// RFC-conventional invalid_target error code.
func TestOAuth21_ResourceIndicator_Validation(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	registerTestClient(t, ts, "res-client", "res-secret")

	authorizeWithResource := func(resource string) *httptest.ResponseRecorder {
		router, cookie := mcpSession(t, ts, []string{"openid", "profile", "email"})
		verifier := uuid.New().String() + uuid.New().String()
		sum := sha256.Sum256([]byte(verifier))
		challenge := base64.RawURLEncoding.EncodeToString(sum[:])
		qs := url.Values{}
		qs.Set("client_id", "res-client")
		qs.Set("response_type", "code")
		qs.Set("redirect_uri", "http://localhost:3000/callback")
		qs.Set("code_challenge", challenge)
		qs.Set("code_challenge_method", "S256")
		qs.Set("state", uuid.New().String())
		qs.Set("response_mode", "query")
		qs.Set("scope", "openid profile email")
		qs.Set("resource", resource)
		return doAuthorizeGET(router, qs, cookie)
	}

	invalidTargetRejected := func(t *testing.T, w *httptest.ResponseRecorder) {
		t.Helper()
		require.Equal(t, http.StatusFound, w.Code, "body: %s", w.Body.String())
		loc, err := url.Parse(w.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "invalid_target", loc.Query().Get("error"),
			"invalid resource must be rejected with invalid_target: %s", w.Header().Get("Location"))
	}

	t.Run("relative reference rejected", func(t *testing.T) {
		invalidTargetRejected(t, authorizeWithResource("/api/resource"))
	})
	t.Run("non-URI string rejected", func(t *testing.T) {
		invalidTargetRejected(t, authorizeWithResource("not-a-uri"))
	})
	t.Run("fragment rejected", func(t *testing.T) {
		invalidTargetRejected(t, authorizeWithResource("https://mcp.example.com/api#frag"))
	})

	t.Run("valid absolute URI accepted", func(t *testing.T) {
		w := authorizeWithResource("https://mcp.example.com/api")
		code := codeFromRedirect(t, w) // redirects with a code, not an error
		require.NotEmpty(t, code)
	})
}

// Item 4 (LOW/MEDIUM): RFC 8707 audience restriction must be enforced at
// Authorizer's OWN protected endpoints. A token minted for an external resource
// (aud = resource URI) must be rejected at /userinfo, while a normal
// client-bound token still works. Introspection keeps its own path and is
// unaffected (see /oauth/introspect — it uses ParseJWTToken directly).
func TestOAuth21_ResourceBoundToken_RejectedAtUserInfo(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := signupUser(t, ts)

	// A resource-bound access token (aud = external resource URI).
	resourceToken := issueAccessTokenWithResource(t, ts, ctx, email, "https://mcp.example.com")
	assert.Equal(t, "https://mcp.example.com", decodeJWTPayloadUnverified(t, resourceToken)["aud"],
		"sanity: token aud must be the resource")
	code, _ := callUserInfo(t, ts, resourceToken)
	assert.Equal(t, http.StatusUnauthorized, code,
		"a resource-bound token must not authenticate at authorizer's own /userinfo")

	// A normal client-bound token (aud = default audience) still works.
	clientToken := issueAccessTokenWithScopes(t, ts, ctx, email, []string{"openid", "profile", "email"})
	code2, body := callUserInfo(t, ts, clientToken)
	assert.Equal(t, http.StatusOK, code2, "a client-bound token must still work at /userinfo: %v", body)
}

// issueAccessTokenWithResource mints and persists an access token whose `aud`
// is the given RFC 8707 resource indicator, mirroring what /oauth/token does
// for a resource-bound exchange.
func issueAccessTokenWithResource(t *testing.T, ts *testSetup, ctx context.Context, email, resource string) string {
	t.Helper()
	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	nonce := "nonce-" + uuid.New().String()
	authToken, err := ts.TokenProvider.CreateAuthToken(nil, &token.AuthTokenConfig{
		User:        user,
		Roles:       []string{"user"},
		Scope:       []string{"openid", "profile", "email"},
		LoginMethod: constants.AuthRecipeMethodBasicAuth,
		Nonce:       nonce,
		HostName:    "http://localhost",
		Resource:    resource,
	})
	require.NoError(t, err)
	require.NotNil(t, authToken.AccessToken)

	sessionStoreKey := constants.AuthRecipeMethodBasicAuth + ":" + user.ID
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		sessionStoreKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt))
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		sessionStoreKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt))
	return authToken.AccessToken.Token
}

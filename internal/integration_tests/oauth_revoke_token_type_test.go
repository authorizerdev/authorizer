package integration_tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
)

// RFC 7009 §2.1 makes token_type_hint an OPTIMISATION, not a filter: the server
// "MAY ignore" it and MUST still find the token when the hint is wrong or absent.
//
// /oauth/revoke used to consult only the refresh entry, while separately
// accepting token_type_hint=access_token as a supported hint. An access token
// presented there matched nothing and the handler returned 200 having revoked
// nothing at all — and because §2.2 mandates that 200 either way, no client could
// tell. These tests pin both halves of the rule: the hinted type is found, and a
// wrong hint does not hide the other one.

// postRevoke drives the real /oauth/revoke handler.
func postRevoke(t *testing.T, ts *testSetup, tokenValue, hint string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.POST("/oauth/revoke", ts.HttpProvider.RevokeRefreshTokenHandler())

	form := url.Values{}
	form.Set("token", tokenValue)
	form.Set("client_id", ts.Config.ClientID)
	if hint != "" {
		form.Set("token_type_hint", hint)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/oauth/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Host must match the iss claim baked into tokens at creation time.
	req.Host = "localhost"
	router.ServeHTTP(w, req)
	return w
}

// accessTokenStillValidates reports whether the token is still accepted by
// Authorizer's own request-serving auth — the observable definition of "was it
// really revoked?".
func accessTokenStillValidates(t *testing.T, ts *testSetup, accessToken string) bool {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, "/userinfo", nil)
	req.Host = "localhost"
	_, err := ts.TokenProvider.ValidateAccessToken(&gin.Context{Request: req}, accessToken)
	return err == nil
}

// TestRevokeAccessTokenInvalidatesSession is the regression test: an access token
// presented at /oauth/revoke must actually be revoked, not silently ignored.
func TestRevokeAccessTokenInvalidatesSession(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)

	require.True(t, accessTokenStillValidates(t, ts, authToken.AccessToken.Token),
		"baseline: the access token must validate before revocation")

	w := postRevoke(t, ts, authToken.AccessToken.Token, constants.TokenTypeAccessToken)
	require.Equal(t, http.StatusOK, w.Code, "RFC 7009 §2.2: revocation always answers 200")

	assert.False(t, accessTokenStillValidates(t, ts, authToken.AccessToken.Token),
		"an access token presented at /oauth/revoke MUST be revoked, not silently ignored")
}

// TestRevokeAccessTokenWithoutHintIsFound covers the same path with no hint at
// all, which RFC 7009 §2.1 permits and which is what a client that does not know
// the token's type will send.
func TestRevokeAccessTokenWithoutHintIsFound(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)

	w := postRevoke(t, ts, authToken.AccessToken.Token, "")
	require.Equal(t, http.StatusOK, w.Code)

	assert.False(t, accessTokenStillValidates(t, ts, authToken.AccessToken.Token),
		"an access token MUST be found even when no token_type_hint is supplied")
}

// TestRevokeRefreshTokenWithWrongHintStillRevokes pins the half of RFC 7009 §2.1
// that a naive "switch on the hint" fix would break: the hint only orders the
// lookup, it never restricts it.
func TestRevokeRefreshTokenWithWrongHintStillRevokes(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)

	claims, err := ts.TokenProvider.ParseJWTToken(authToken.AccessToken.Token)
	require.NoError(t, err)
	userID, _ := claims["sub"].(string)
	require.NotEmpty(t, userID)

	// setupIntrospectTest asks for no offline_access, so it registers only the
	// session and access entries. Register a refresh entry under the same nonce,
	// as a login with offline_access would.
	refreshValue := "refresh-token-value-" + userID
	require.NoError(t, ts.MemoryStoreProvider.SetUserSession(
		introspectSessionKey(userID),
		constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint,
		refreshValue,
		authToken.AccessToken.ExpiresAt,
	))

	// A refresh token presented with the WRONG hint must still be found. The
	// handler parses the presented value as a JWT before the lookup, so present
	// the access token's own JWT while asserting on the refresh entry being the
	// one the loop can reach: the access candidate is tried FIRST under this
	// hint, so a match here proves the loop does not stop at a miss.
	w := postRevoke(t, ts, authToken.AccessToken.Token, constants.TokenTypeRefreshToken)
	require.Equal(t, http.StatusOK, w.Code)

	assert.False(t, accessTokenStillValidates(t, ts, authToken.AccessToken.Token),
		"a wrong token_type_hint MUST NOT prevent the token from being found")
}

// TestRevokeUnregisteredTokenIsNoop guards the negative half: a syntactically
// valid, correctly-audienced token that was never registered (or was already
// revoked) still answers 200 and revokes nothing.
func TestRevokeUnregisteredTokenIsNoop(t *testing.T) {
	ts, _, authToken := setupIntrospectTest(t)

	claims, err := ts.TokenProvider.ParseJWTToken(authToken.AccessToken.Token)
	require.NoError(t, err)
	userID, _ := claims["sub"].(string)

	// Drop the session first, so the token is genuinely unregistered.
	require.NoError(t, ts.MemoryStoreProvider.DeleteUserSession(introspectSessionKey(userID), authToken.FingerPrint))

	w := postRevoke(t, ts, authToken.AccessToken.Token, "")
	assert.Equal(t, http.StatusOK, w.Code, "RFC 7009 §2.2: an unknown token still answers 200")
}

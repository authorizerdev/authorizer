package integration_tests

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RFC 6749 §2.3.1: client_id and client_secret carried in the HTTP Basic
// Authorization header MUST each be application/x-www-form-urlencoded before
// being placed there. Some client libraries (e.g. Java's URLEncoder, used by
// the OpenID Foundation conformance suite) percent-encode characters like "!"
// under that algorithm. A server that only base64-decodes the header — never
// undoing that encoding — rejects a perfectly valid secret containing such a
// character. Regression test for a real conformance failure (oidcc-refresh-
// token, "second client" token exchange) caused by a client secret containing
// "!": the raw HTTP Basic password arrived as "...%21" instead of "...!".
func TestTokenClientAuth_BasicAuth_FormURLEncodedSecret_Accepted(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	registerTestClient(t, ts, "client-bang", "Secret!With*Special'Chars")

	router := gin.New()
	router.POST("/oauth/token", ts.HttpProvider.TokenHandler())

	// application/x-www-form-urlencoded (RFC 3986 percent-encoding, the
	// legacy convention Java's URLEncoder follows) of "Secret!With*Special'Chars"
	// percent-encodes "!", "*", and "'".
	encodedSecret := "Secret%21With%2ASpecial%27Chars"
	basicAuth := base64.StdEncoding.EncodeToString([]byte("client-bang:" + encodedSecret))

	req, err := http.NewRequest(http.MethodPost, "/oauth/token", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Basic "+basicAuth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// grant_type is intentionally invalid — this test only proves client
	// authentication succeeds (unsupported_grant_type, not invalid_client).
	req.Body = http.NoBody

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusUnauthorized, w.Code, "body: %s", w.Body.String())
	assert.NotContains(t, w.Body.String(), "invalid_client",
		"a form-urlencoded Basic-auth secret must decode back to the stored secret before comparison")
}

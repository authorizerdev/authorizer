package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProtectedResourceMetadata pins the RFC 9728 document that starts the whole
// MCP discovery chain.
//
// A client that has never seen this deployment learns everything from here: which
// URI to name as its RFC 8707 `resource` (and therefore what audience its token
// will carry), and which authorization server to go to. Get `resource` wrong and
// every token a client obtains is bound to an audience /mcp will reject; get
// `authorization_servers` wrong and the client never reaches the OAuth flow at
// all. Both failures look identical from outside — a permanent 401 — which is why
// the exact strings are asserted rather than merely their presence.
func TestProtectedResourceMetadata(t *testing.T) {
	cfg := getTestConfig()
	cfg.AuthorizerURL = "https://auth.example.com"
	cfg.MCPEnabled = true
	ts := initTestSetup(t, cfg)

	router := gin.New()
	// ONE path, matching the real router. RFC 9728 §3.1 inserts the well-known
	// segment ahead of the resource identifier's PATH, so this URL denotes
	// "<url>/mcp"; the bare well-known path denotes the origin, and §3.3 has
	// clients reject a document whose `resource` does not match the identifier
	// they used to build the request.
	router.GET("/.well-known/oauth-protected-resource/mcp", ts.HttpProvider.ProtectedResourceMetadataHandler())

	get := func(t *testing.T, path string) map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		// A hostile Host/X-Authorizer-URL must not change the advertised
		// resource: the document is derived from --url alone. If it could be
		// steered, an attacker could publish a resource identifier matching a
		// token they already hold.
		req.Host = "evil.example.com"
		req.Header.Set("X-Authorizer-URL", "https://evil.example.com")
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

		var doc map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &doc))
		return doc
	}

	t.Run("the document is correct at the RFC 9728 §3.1 path", func(t *testing.T) {
		doc := get(t, "/.well-known/oauth-protected-resource/mcp")
		assert.Equal(t, "https://auth.example.com/mcp", doc["resource"],
			"this is the exact string clients send as `resource` and tokens carry as `aud`")
		assert.Equal(t, []any{"https://auth.example.com"}, doc["authorization_servers"],
			"RFC 9728 requires at least one authorization server; here Authorizer is its own")
		assert.Equal(t, []any{"header"}, doc["bearer_methods_supported"],
			"MCP forbids tokens in the query string — only the Authorization header")
		assert.Contains(t, doc["scopes_supported"], "offline_access",
			"a client that requests only what this document advertises must still be able to obtain a refresh token")
	})
}

// TestProtectedResourceMetadataNormalizesIssuer pins the one thing that has to
// agree across three documents: the origin.
//
// `resource` is normalized (scheme+host) and so is the `iss` claim on every
// token, because both come from the same sanitizer parsers.GetHost uses. If the
// advertised authorization server and jwks_uri were built from the RAW --url
// instead, an operator whose --url carries a path or a trailing slash would
// publish an authorization server that 404s and an issuer no token matches,
// while startup reported everything fine — a discovery chain that dead-ends with
// no error anywhere.
func TestProtectedResourceMetadataNormalizesIssuer(t *testing.T) {
	for _, rawURL := range []string{
		"https://auth.example.com/auth", // a path an operator might add behind a proxy
		"https://auth.example.com/",     // a trailing slash
	} {
		t.Run(rawURL, func(t *testing.T) {
			cfg := getTestConfig()
			cfg.AuthorizerURL = rawURL
			cfg.MCPEnabled = true
			ts := initTestSetup(t, cfg)

			router := gin.New()
			router.GET("/.well-known/oauth-protected-resource/mcp", ts.HttpProvider.ProtectedResourceMetadataHandler())

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/mcp", nil))
			require.Equal(t, http.StatusOK, w.Code)

			var doc map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &doc))
			assert.Equal(t, "https://auth.example.com/mcp", doc["resource"])
			assert.Equal(t, []any{"https://auth.example.com"}, doc["authorization_servers"])
			assert.Equal(t, "https://auth.example.com/.well-known/jwks.json", doc["jwks_uri"])
		})
	}
}

// TestProtectedResourceMetadataFailsClosedWithoutURL asserts the handler refuses
// to emit a document when no canonical URL is configured, rather than publishing
// one with an empty `resource`.
//
// Startup already refuses --mcp-enabled without --url, so this is the second lock
// on the same door. It is worth having because the failure it prevents is silent:
// a document advertising resource:"" would lead clients to request tokens with an
// empty audience, which is exactly the audience an unconfigured deployment would
// then be comparing against.
func TestProtectedResourceMetadataFailsClosedWithoutURL(t *testing.T) {
	cfg := getTestConfig()
	cfg.AuthorizerURL = ""
	cfg.MCPEnabled = true
	ts := initTestSetup(t, cfg)

	router := gin.New()
	router.GET("/.well-known/oauth-protected-resource", ts.HttpProvider.ProtectedResourceMetadataHandler())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

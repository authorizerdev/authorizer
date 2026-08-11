package http_handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ProtectedResourceMetadataHandler serves OAuth 2.0 Protected Resource Metadata
// (RFC 9728) for this deployment's MCP server.
//
// The MCP authorization spec makes this mandatory — "MCP servers MUST implement
// OAuth 2.0 Protected Resource Metadata" — and it is the entry point of the whole
// discovery chain: an unauthenticated call to /mcp returns 401 with a
// WWW-Authenticate header pointing here, the client reads `authorization_servers`
// from this document, fetches the AS metadata from there, and only then starts the
// OAuth flow. Without this endpoint an MCP client has no way to learn where to
// authenticate, so the surface is unreachable no matter how correct the rest is.
//
// Served at ONE path: /.well-known/oauth-protected-resource/mcp.
//
// RFC 9728 §3.1 builds the metadata URL by inserting the well-known segment
// between the host and the PATH of the resource identifier, so this URL is the
// one that denotes "https://<host>/mcp". The bare
// /.well-known/oauth-protected-resource denotes a different identifier —
// "https://<host>", the origin with no path — and §3.3 requires a client to
// reject a document whose `resource` is not identical to the identifier it used
// to build the request. Serving this document there too would hand strict
// clients a mismatch to reject. Clients that probe the origin form are expected
// to reach us the documented way instead: the 401 from /mcp carries
// `WWW-Authenticate: Bearer resource_metadata="<this URL>"`, which §5.1 makes
// the primary discovery mechanism, and well-known probing only the fallback.
//
// Here Authorizer is both the resource server and the authorization server, which
// is the easy case: `authorization_servers` names this same origin, so there is no
// third party to be confused about and no token to pass through to one.
//
// # Which clients can actually complete this flow today
//
// Verified against Claude Code 2.1.226: the browser OAuth path does NOT work. It
// refuses with "Incompatible auth server: does not support dynamic client
// registration" and does not fall back to prompting for a client id. Everything
// this document advertises is correct and the flow is spec-conformant — the gap
// is client-side registration, not discovery.
//
// What IS verified working is a static bearer token bound to <url>/mcp, minted
// via client_credentials with `resource`. That identifies a service account
// rather than a human, which is the honest scope of the surface until Client ID
// Metadata Documents land. See the note on `registration_endpoint` in
// openid_config.go for why the answer is CIMD and not RFC 7591 DCR.
//
// Deliberately public and cacheable: RFC 9728 §3.1 defines this as public client
// configuration. It carries no secret and no per-caller data — only where to
// authenticate and what to ask for.
func (h *httpProvider) ProtectedResourceMetadataHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Both derive from Config.CanonicalURL, which is the same normalization
		// parsers.GetHost applies. The issuer advertised here therefore matches
		// the `iss` claim on every token this server mints and the base of every
		// URL the OIDC discovery document publishes. Reading the raw
		// Config.AuthorizerURL instead would diverge the moment an operator's
		// --url carried a path or a trailing slash — the metadata would name an
		// authorization server that 404s while startup reported success.
		base := h.Config.CanonicalURL()
		resource := h.Config.MCPResource()
		if base == "" || resource == "" {
			// Unreachable in practice: the route is only registered when MCP is
			// enabled, and startup refuses --mcp-enabled without --url. Answering
			// 404 rather than emitting a document with an empty `resource` keeps
			// the failure honest if that invariant ever breaks — a client that
			// cannot discover us is far better than one that binds its tokens to
			// an empty audience.
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"resource":              resource,
			"authorization_servers": []string{base},
			// RFC 6750 §2.1 header form only. MCP forbids the token in the query
			// string, and this server never reads it from a form body.
			"bearer_methods_supported": []string{"header"},
			// Shared with the authorization server metadata rather than restated:
			// a client that asks for exactly what this document advertises and
			// gets no `offline_access` receives no refresh token, and the agent's
			// session dies at access-token expiry.
			"scopes_supported": supportedScopes,
			// Clients use this to fetch the signing keys when they want to
			// inspect a token locally; harmless to advertise and saves a probe.
			"jwks_uri":               base + "/.well-known/jwks.json",
			"resource_documentation": "https://docs.authorizer.dev/core/mcp",
		})
	}
}

// MCPAuthMiddleware authenticates a request to /mcp and, when it cannot, returns
// the RFC 9728 §5.1 challenge that starts the discovery chain.
//
// The 401 is not merely an error, it is the protocol. A client with no
// credential reads `resource_metadata` from WWW-Authenticate, fetches the
// metadata document, finds the authorization server and begins OAuth; a client
// whose token has expired reads the same 401 and refreshes. Anthropic's connector
// documentation is explicit that Claude refreshes reactively on a 401 and does
// not honour a WWW-Authenticate header on a 200 — so answering an expired token
// with a JSON-RPC error inside a 200 would leave the client looping on a dead
// token with no way to discover why.
//
// This authenticates a SECOND time: the in-process gRPC interceptor resolves the
// identity that handlers actually run under (interceptors.MCPTokenResolver), and
// context values do not survive the bufconn hop, so the check cannot be shared.
// The duplication buys the correct HTTP status, which is what makes the surface
// usable by a real client. It costs one extra token validation per request —
// worth optimising later, not worth trading the protocol for.
func (h *httpProvider) MCPAuthMiddleware() gin.HandlerFunc {
	// Computed once: both derive from --url, which MCP requires, so neither can
	// be influenced by the request.
	resource := h.Config.MCPResource()
	metadataURL := h.Config.CanonicalURL() + "/.well-known/oauth-protected-resource/mcp"

	return func(c *gin.Context) {
		log := h.Log.With().Str("func", "MCPAuthMiddleware").Logger()

		// RFC 6750 §3: a request with NO credential gets a bare challenge. An
		// `error` parameter is only correct once a credential was supplied and
		// found wanting — reporting invalid_token to a client that sent nothing
		// misdescribes a first contact as a failure.
		accessToken, err := h.TokenProvider.GetAccessToken(c)
		if err != nil || accessToken == "" {
			c.Header("WWW-Authenticate",
				`Bearer realm="authorizer", resource_metadata="`+metadataURL+`"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_request",
				"error_description": "authorization required",
			})
			return
		}

		if _, vErr := h.TokenProvider.ValidateMCPAccessToken(c, accessToken, resource); vErr != nil {
			// Debug, not warn: a wrong-audience or expired token at this endpoint
			// is the ordinary steady state of a client that needs to refresh, not
			// evidence of an attack.
			log.Debug().Err(vErr).Msg("mcp request rejected")
			c.Header("WWW-Authenticate",
				`Bearer realm="authorizer", error="invalid_token", `+
					`error_description="The access token is invalid, expired, or was not issued for this MCP server", `+
					`resource_metadata="`+metadataURL+`"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "The access token is invalid, expired, or was not issued for this MCP server",
			})
			return
		}

		c.Next()
	}
}

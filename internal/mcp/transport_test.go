package mcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandlerIsStateless pins the two transport options the HTTP surface depends
// on, by observing their effects rather than reading the struct back.
//
// This replaces TestServer_StdioOnly, which existed to stop a network transport
// being added before MCP had an auth layer, and which named its own exit
// condition: "implement an auth+rate-limit interceptor for MCP first, then
// update this test's allow-list." That interceptor now exists
// (interceptors.MCPTokenResolver plus the sole-authority guard), so the guard is
// replaced rather than deleted — the property worth protecting is no longer
// "there is no HTTP transport" but "the HTTP transport is the stateless,
// non-streaming shape the main listener can actually serve".
//
// Why each matters:
//
//   - GET must be refused. The main HTTP server sets WriteTimeout: 60s, so a
//     long-lived SSE stream would be severed mid-flight with no error the client
//     can distinguish from a network fault. A 405 tells the client immediately
//     that this server does not offer a server→client stream.
//   - No server-side session state. A stateful server pins a client to the
//     replica that created its session, which would make the MCP surface the one
//     part of Authorizer that cannot scale horizontally without sticky sessions.
//     Asserted by sending an Mcp-Session-Id this process has never issued and
//     requiring it to be served anyway — the observable form of "any replica can
//     answer any request". Asserting on the response's session-id header instead
//     would pin an SDK detail: in stateless mode an id may still be emitted, it
//     is simply never validated.
func TestHandlerIsStateless(t *testing.T) {
	// A real inner MCP server: the handler answers 400 "no server available"
	// without one, which would make every assertion below vacuous.
	s := &Server{mcpSrv: mcp.NewServer(&mcp.Implementation{Name: "authorizer", Version: "test"}, nil)}

	t.Run("GET is refused with an Allow header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		// Without this the SDK rejects the GET at Accept negotiation, before it
		// ever reaches the stateless branch, and the assertion would pass for the
		// wrong reason.
		req.Header.Set("Accept", "text/event-stream")
		s.Handler().ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code,
			"a stateless server must not offer an SSE stream the 60s write timeout would kill")
		assert.Equal(t, "POST", w.Header().Get("Allow"), "RFC 9110 §15.5.6: a 405 MUST carry Allow")
	})

	t.Run("a request bearing an unknown session id is still served", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(initializeBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Mcp-Session-Id", "issued-by-some-other-replica")
		s.Handler().ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code,
			"a session this replica never created must not be rejected, or the surface needs sticky sessions; body: %s", w.Body.String())
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json",
			"JSONResponse: one JSON response per POST, not an event stream")
	})
}

// initializeBody is the MCP handshake every client sends first.
const initializeBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{` +
	`"protocolVersion":"2025-06-18","capabilities":{},` +
	`"clientInfo":{"name":"t","version":"1"}}}`

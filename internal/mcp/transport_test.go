package mcp

import (
	"reflect"
	"strings"
	"testing"
)

// TestServer_StdioOnly is a guard against accidentally adding a non-stdio
// transport to the MCP server. Stdio is the only supported transport — the
// security model relies on the OS-level trust boundary of the subprocess
// (Claude Code spawns `authorizer mcp` as a child; only that process can
// write to its stdin). Exposing MCP over TCP/HTTP/SSE without an auth
// interceptor would be a security regression, so this test fails the build
// if anyone adds RunHTTP / RunTCP / RunSSE / Listen* / Serve* etc.
//
// To deliberately add a new transport: implement an auth+rate-limit
// interceptor for MCP first, then update this test's allow-list.
func TestServer_StdioOnly(t *testing.T) {
	allowed := map[string]struct{}{
		"RunStdio":  {},
		"MCPServer": {}, // test accessor — not a transport
	}
	t.Logf("MCP Server exported methods allow-list: %v (anything outside this set indicates a new transport)", allowed)

	st := reflect.TypeOf((*Server)(nil))
	for i := 0; i < st.NumMethod(); i++ {
		name := st.Method(i).Name
		if _, ok := allowed[name]; ok {
			continue
		}
		// Heuristic: any method whose name suggests serving / running /
		// listening over a different transport is a red flag.
		lower := strings.ToLower(name)
		for _, banned := range []string{"http", "tcp", "sse", "websocket", "listen", "serve", "run"} {
			if strings.Contains(lower, banned) {
				t.Errorf("disallowed transport method %q on *Server: stdio is the only supported MCP transport. "+
					"Adding a network transport requires an MCP-side auth interceptor first; see Server type comment.", name)
			}
		}
	}
}

package mcp

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// identityFreeExposedMethods are the MCP-exposed RPCs that are ALSO marked
// `public` and are therefore allowed to be so.
//
// Adding a name here is a security assertion, not a formality: it claims the
// method never resolves a caller identity — no authctx.Principal read, no
// callerTokenData, no resolveFgaCaller, no direct TokenProvider use. Verify that
// in the service implementation before adding one.
var identityFreeExposedMethods = map[string]string{
	// internal/service/meta.go returns static deployment configuration and
	// contains no reference to authctx, callerTokenData, resolveFgaCaller or
	// TokenProvider.
	"Meta": "returns static deployment configuration; resolves no caller",
}

// TestExposedMCPToolsCannotBypassTheMCPTokenRule guards the assumption the MCP
// audience boundary actually rests on.
//
// The boundary is enforced by giving the MCP surface its own gRPC server whose
// auth interceptor uses MCPTokenResolver — a token is accepted only when its
// `aud` names this MCP server. But the interceptor attaches a principal only for
// methods that require auth. For a method marked `public` it calls the handler
// with no principal at all, and the service layer then resolves the caller
// itself: callerTokenData (internal/service/caller.go:30) and resolveFgaCaller
// (internal/service/fga.go) both fall back to
// TokenProvider.GetUserIDFromSessionOrAccessToken, which is the DEFAULT rule —
// the one that accepts an ordinary client_id-audience login token and rejects
// the resource-bound audience every MCP token carries.
//
// So a method that is both `mcp_tool.exposed` and `public` and that
// opportunistically resolves an identity would authenticate MCP callers with a
// token that never named the MCP server, silently, on an internet-facing,
// CSRF-exempt endpoint. Nothing would fail to compile and no other test covers
// it: today the property holds only because the single method in that
// intersection is Meta, which resolves nobody.
//
// This test turns that coincidence into a checked invariant. It is the same
// shape as TestAdminMethodsAreGated and TestNotFoundContractIsUniform: a static
// assertion over annotations, so a one-line proto change cannot quietly move the
// security boundary.
func TestExposedMCPToolsCannotBypassTheMCPTokenRule(t *testing.T) {
	var checked int
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		svcs := fd.Services()
		for i := 0; i < svcs.Len(); i++ {
			methods := svcs.Get(i).Methods()
			for j := 0; j < methods.Len(); j++ {
				m := methods.Get(j)
				tool := mcpToolFromMethod(m)
				if tool == nil || !tool.GetExposed() {
					continue
				}
				checked++
				if !methodIsPublic(m) {
					// Requires auth, so the interceptor resolves identity through
					// this server's own resolver and attaches a principal. Safe.
					continue
				}
				name := string(m.Name())
				if _, ok := identityFreeExposedMethods[name]; !ok {
					t.Errorf("RPC %s is both (authorizer.v1.mcp_tool).exposed and (authorizer.v1.public), "+
						"so the MCP server's auth interceptor will invoke it with NO principal and its "+
						"service implementation would resolve the caller with the DEFAULT token rule — "+
						"accepting a login token that never named the MCP server as its audience. "+
						"Either drop the `public` annotation, or (only if the method resolves no caller "+
						"identity at all) add it to identityFreeExposedMethods with the reason.",
						m.FullName())
				}
			}
		}
		return true
	})

	if checked == 0 {
		t.Fatal("found no mcp_tool-exposed methods — the proto registry was not linked in, so this test proved nothing")
	}
	t.Logf("checked %d mcp_tool-exposed methods", checked)
}

// methodIsPublic mirrors interceptors.isPublicMethod. Duplicated rather than
// exported across packages because it is two lines and this test must read the
// annotation exactly as the interceptor does; a shared helper that drifted from
// the interceptor would make this test agree with itself instead of with the
// code it guards.
func methodIsPublic(m protoreflect.MethodDescriptor) bool {
	opts := m.Options()
	if opts == nil {
		return false
	}
	switch v := proto.GetExtension(opts, authorizerv1.E_Public).(type) {
	case bool:
		return v
	case *bool:
		return v != nil && *v
	}
	return false
}

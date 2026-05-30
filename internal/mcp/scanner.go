// Package mcp exposes a subset of Authorizer's gRPC methods as MCP tools.
// Which methods are exposed is declared at the proto layer via the custom
// option `authorizer.common.v1.mcp_tool` — the scanner reads it at startup
// to build the tool registry. No service-by-service hand-registration.
package mcp

import (
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	commonv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/common/v1"
)

// ToolBinding is one MCP-exposed RPC: a tool name, plus enough metadata to
// dispatch a JSON-arg invocation back to the gRPC server.
type ToolBinding struct {
	// Name surfaced to MCP clients (e.g. "get_meta"). Defaults to
	// snake_case(method) unless the proto annotation overrides it.
	Name string
	// Description from the RPC's leading comment, surfaced to the MCP host.
	Description string
	// Destructive hints to the MCP host that user confirmation is warranted.
	Destructive bool

	// FullMethod is the gRPC method name in `/pkg.Service/Method` form.
	// Used directly with grpc.ClientConn.Invoke.
	FullMethod string
	// InputDescriptor / OutputDescriptor are the proto message descriptors
	// for request/response. Used by the dispatcher to construct dynamic
	// proto.Message instances for JSON unmarshalling/marshalling.
	InputDescriptor  protoreflect.MessageDescriptor
	OutputDescriptor protoreflect.MessageDescriptor
}

// Scan walks the supplied gRPC server's registered services and returns the
// set of methods marked `(authorizer.common.v1.mcp_tool).exposed = true`.
// Methods that aren't exposed (the default) are silently skipped.
func Scan(srv *grpc.Server) ([]ToolBinding, error) {
	var bindings []ToolBinding
	for svcName := range srv.GetServiceInfo() {
		// Look up the proto descriptor for this service by full name.
		desc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(svcName))
		if err != nil {
			// Not all gRPC services come from compiled proto (e.g. the
			// gRPC health checking and reflection services). Skip silently.
			continue
		}
		svcDesc, ok := desc.(protoreflect.ServiceDescriptor)
		if !ok {
			continue
		}

		methods := svcDesc.Methods()
		for i := 0; i < methods.Len(); i++ {
			m := methods.Get(i)
			tool := mcpToolFromMethod(m)
			if tool == nil || !tool.Exposed {
				continue
			}

			name := tool.ToolName
			if name == "" {
				name = camelToSnake(string(m.Name()))
			}

			bindings = append(bindings, ToolBinding{
				Name:             name,
				Description:      strings.TrimSpace(string(m.ParentFile().SourceLocations().ByDescriptor(m).LeadingComments)),
				Destructive:      tool.Destructive,
				FullMethod:       fmt.Sprintf("/%s/%s", svcName, m.Name()),
				InputDescriptor:  m.Input(),
				OutputDescriptor: m.Output(),
			})
		}
	}
	return bindings, nil
}

// mcpToolFromMethod reads the (authorizer.common.v1.mcp_tool) option off a
// method descriptor. Returns nil when the option is absent or unset.
func mcpToolFromMethod(m protoreflect.MethodDescriptor) *commonv1.McpTool {
	opts := m.Options()
	if opts == nil {
		return nil
	}
	t, ok := proto.GetExtension(opts, commonv1.E_McpTool).(*commonv1.McpTool)
	if !ok || t == nil {
		return nil
	}
	return t
}

// camelToSnake converts MixedCase / camelCase to snake_case. ASCII only;
// proto method names never contain non-ASCII.
func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

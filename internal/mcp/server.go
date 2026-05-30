package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

const bufSize = 1 << 20

// Server wraps an MCP server that bridges to an in-process gRPC server. The
// gRPC server is the source of truth for which tools exist (via the
// `mcp_tool` proto annotation); we never hand-register tools here.
type Server struct {
	log     *zerolog.Logger
	mcpSrv  *mcp.Server
	gwConn  *grpc.ClientConn
	lis     *bufconn.Listener
	grpcSrv *grpc.Server
}

// New builds an MCP server that exposes every gRPC method on `grpcSrv`
// whose proto annotation has `(authorizer.common.v1.mcp_tool).exposed = true`.
// The gRPC server is served over an in-process bufconn — same pattern as
// the REST gateway — so MCP tool invocations become local method calls with
// no extra network hop.
func New(log *zerolog.Logger, grpcSrv *grpc.Server, name, version string) (*Server, error) {
	bindings, err := Scan(grpcSrv)
	if err != nil {
		return nil, fmt.Errorf("mcp: scan tools: %w", err)
	}
	log.Info().Int("tools", len(bindings)).Msg("MCP: discovered tools from proto annotations")

	// Same bufconn dance as the REST gateway.
	lis := bufconn.Listen(bufSize)
	go func() { _ = grpcSrv.Serve(lis) }()
	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		_ = lis.Close()
		return nil, fmt.Errorf("mcp: dial in-process grpc: %w", err)
	}

	mcpSrv := mcp.NewServer(&mcp.Implementation{
		Name:    name,
		Version: version,
	}, nil)

	for _, b := range bindings {
		registerTool(log, mcpSrv, conn, b)
	}

	return &Server{
		log:     log,
		mcpSrv:  mcpSrv,
		gwConn:  conn,
		lis:     lis,
		grpcSrv: grpcSrv,
	}, nil
}

// MCPServer exposes the underlying *mcp.Server. Used by tests to drive the
// server with an in-memory transport pair.
func (s *Server) MCPServer() *mcp.Server { return s.mcpSrv }

// RunStdio serves MCP over stdio (the default Claude Code transport). Blocks
// until ctx is cancelled or the client disconnects.
func (s *Server) RunStdio(ctx context.Context) error {
	defer s.cleanup()
	return s.mcpSrv.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) cleanup() {
	_ = s.gwConn.Close()
	_ = s.lis.Close()
}

// registerTool wires one ToolBinding into the MCP server. The handler:
//   1. Constructs a fresh proto.Message of the right type via dynamicpb
//   2. Unmarshals JSON args into it
//   3. Invokes the gRPC method via grpc.ClientConn.Invoke
//   4. Marshals the response back to JSON for the MCP client
func registerTool(log *zerolog.Logger, srv *mcp.Server, conn *grpc.ClientConn, b ToolBinding) {
	schema := schemaForMessage(b.InputDescriptor)
	tool := &mcp.Tool{
		Name:        b.Name,
		Description: b.Description,
		InputSchema: schema,
	}
	if b.Destructive {
		// MCP clients show a destructive-action confirmation when this is set.
		tool.Annotations = &mcp.ToolAnnotations{DestructiveHint: ptrTrue()}
	}

	srv.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Build a dynamic proto.Message for the request, then unmarshal JSON.
		reqMsg := dynamicpb.NewMessage(b.InputDescriptor)
		if len(req.Params.Arguments) > 0 && !isJSONNull(req.Params.Arguments) {
			if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(req.Params.Arguments, reqMsg); err != nil {
				// Argument decode failures surface as tool errors (not
				// protocol errors) so the LLM gets actionable text.
				return errorResult("invalid arguments: " + err.Error()), nil
			}
		}

		respMsg := dynamicpb.NewMessage(b.OutputDescriptor)
		if err := conn.Invoke(ctx, b.FullMethod, reqMsg, respMsg); err != nil {
			log.Debug().Err(err).Str("tool", b.Name).Str("method", b.FullMethod).Msg("MCP tool invocation failed")
			// gRPC errors (Unimplemented, PermissionDenied, NotFound, ...)
			// become CallToolResult{IsError: true} with the gRPC status
			// message as the content. The MCP host shows this to the LLM
			// in a way that lets it react / try a different tool, rather
			// than a low-level JSON-RPC failure that would just abort.
			return errorResult(err.Error()), nil
		}

		respJSON, err := (protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}).Marshal(respMsg)
		if err != nil {
			return errorResult("encode response: " + err.Error()), nil
		}
		// Surface as both Content (text-shaped) and StructuredContent so MCP
		// clients that prefer either get something they can consume.
		var structured any
		_ = json.Unmarshal(respJSON, &structured)
		return &mcp.CallToolResult{
			Content:           []mcp.Content{&mcp.TextContent{Text: string(respJSON)}},
			StructuredContent: structured,
		}, nil
	})
}

func ptrTrue() *bool { v := true; return &v }

// errorResult wraps a message as a CallToolResult with IsError set. This is
// the MCP-spec way to tell the host that the tool *ran* but produced a
// recoverable error (vs the JSON-RPC-level error path which signals a
// protocol/transport failure).
func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}

// isJSONNull returns true when the raw JSON encodes a literal `null`, with
// any surrounding whitespace tolerated.
func isJSONNull(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s == "null"
}

// compile-time assertion that ToolBinding messages descriptors implement what we need.
var _ proto.Message = (*dynamicpb.Message)(nil)

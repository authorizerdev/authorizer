// Package mcp serves a curated subset of Authorizer's gRPC methods to
// LLM clients via the Model Context Protocol. Stdio is the ONLY supported
// transport — see the deliberate design note on Server below.
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
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

const bufSize = 1 << 20

// Server wraps an MCP server that bridges to an in-process gRPC server.
//
// Design constraint: stdio is the ONLY supported transport. The MCP server
// has no auth/rate-limit/audit interceptors of its own — it relies entirely
// on the OS-level trust boundary of the subprocess (Claude Code spawns
// `authorizer mcp` as a child; only that process can write to its stdin).
// Exposing the MCP server over TCP / HTTP / SSE would invalidate that
// assumption and is intentionally NOT implementable: there is no RunHTTP /
// RunTCP / RunSSE method, and adding one without first implementing an
// auth layer is a security regression. The stdio-only contract is also
// enforced by TestServer_StdioOnly.
type Server struct {
	log     *zerolog.Logger
	mcpSrv  *mcp.Server
	gwConn  *grpc.ClientConn
	lis     *bufconn.Listener
	grpcSrv *grpc.Server

	// bearer is the value of the Authorization header stamped on every
	// outgoing gRPC call. Set via Options.Bearer at construction time
	// (the cmd/mcp.go subcommand exposes --mcp-bearer). When empty, calls
	// flow without auth — fine for public methods like Meta, but anything
	// requiring identity (Profile, CheckPermissions, ...) will see an empty
	// caller and return whatever its handler does in that case.
	bearer string
	// authorizerURL is stamped as `x-authorizer-url` metadata on every
	// outgoing gRPC call. JWT issuer validation compares a token's `iss`
	// against the resolved host; the in-process bufconn call would resolve
	// to "http://bufconn", so without this every bearer token minted by the
	// real server would be rejected. Set it to the public URL of the
	// Authorizer instance that issued the bearer token.
	authorizerURL string
}

// Options configures the MCP server.
type Options struct {
	// Name is the MCP server's reported implementation name.
	Name string
	// Version is the MCP server's reported implementation version.
	Version string
	// Bearer, when set, is propagated as `Authorization: Bearer <value>`
	// metadata on every gRPC dispatch. This is how MCP-side identity
	// reaches the gRPC handlers (security audit H1). The bearer should be
	// a token issued for the user the MCP host is acting on behalf of.
	Bearer string
	// AuthorizerURL, when set, is propagated as `x-authorizer-url` metadata
	// on every gRPC dispatch so JWT issuer validation resolves the host the
	// bearer token was minted by (not the in-process "bufconn" authority).
	// Required for identity-bearing tools when Bearer is set.
	AuthorizerURL string
}

// New builds an MCP server that exposes every gRPC method on `grpcSrv`
// whose proto annotation has `(authorizer.common.v1.mcp_tool).exposed = true`.
// The gRPC server is served over an in-process bufconn — same pattern as
// the REST gateway — so MCP tool invocations become local method calls with
// no extra network hop.
func New(log *zerolog.Logger, grpcSrv *grpc.Server, opts Options) (*Server, error) {
	bindings, err := Scan(grpcSrv)
	if err != nil {
		return nil, fmt.Errorf("mcp: scan tools: %w", err)
	}
	log.Info().
		Int("tools", len(bindings)).
		Bool("authenticated", opts.Bearer != "").
		Msg("MCP: discovered tools from proto annotations")

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
		Name:    opts.Name,
		Version: opts.Version,
	}, nil)

	s := &Server{
		log:     log,
		mcpSrv:        mcpSrv,
		gwConn:        conn,
		lis:           lis,
		grpcSrv:       grpcSrv,
		bearer:        opts.Bearer,
		authorizerURL: opts.AuthorizerURL,
	}
	for _, b := range bindings {
		s.registerTool(b)
	}
	return s, nil
}

// MCPServer exposes the underlying *mcp.Server. Used by tests to drive the
// server with an in-memory transport pair.
func (s *Server) MCPServer() *mcp.Server { return s.mcpSrv }

// RunStdio serves MCP over stdio (the default Claude Code transport). Blocks
// until ctx is cancelled or the client disconnects.
//
// This is the only `Run*` method on the Server. See the type comment for why
// adding a non-stdio transport is intentionally a code-level non-feature.
func (s *Server) RunStdio(ctx context.Context) error {
	defer s.cleanup()
	return s.mcpSrv.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) cleanup() {
	_ = s.gwConn.Close()
	_ = s.lis.Close()
}

// stampAuth attaches the configured bearer and authorizer URL to the
// outgoing gRPC call. A no-op when neither is set. This is the bridge that
// lets gRPC handlers see "who is calling" (security audit H1) and which
// host minted the token (issuer validation) when invoked from MCP.
func (s *Server) stampAuth(ctx context.Context) context.Context {
	if s.bearer != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+s.bearer)
	}
	if s.authorizerURL != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-authorizer-url", s.authorizerURL)
	}
	return ctx
}

// registerTool wires one ToolBinding into the MCP server. The handler:
//  1. Constructs a fresh proto.Message of the right type via dynamicpb
//  2. Unmarshals JSON args into it
//  3. Invokes the gRPC method via grpc.ClientConn.Invoke (with bearer)
//  4. Marshals the response back to JSON for the MCP client
func (s *Server) registerTool(b ToolBinding) {
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

	s.mcpSrv.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
		if err := s.gwConn.Invoke(s.stampAuth(ctx), b.FullMethod, reqMsg, respMsg); err != nil {
			s.log.Debug().Err(err).Str("tool", b.Name).Str("method", b.FullMethod).Msg("MCP tool invocation failed")
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

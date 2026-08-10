// Package mcp serves a curated subset of Authorizer's gRPC methods to LLM
// clients via the Model Context Protocol, over Streamable HTTP (the deployable
// transport) or stdio (development only). See the design note on Server.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
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
// Two transports, with very different security models.
//
// Handler() serves Streamable HTTP and is the deployable one. It carries no
// ambient authority: every tool call is authenticated by the caller's own bearer
// token, whose audience must name this MCP server (token.ValidateMCPAccessToken),
// and it runs on a gRPC server whose auth interceptor accepts nothing else — no
// cookies, no admin secret, no admin service. The route wrapper rejects a bad
// credential with a 401 so clients can start discovery or refresh.
//
// RunStdio has no auth of its own and relies entirely on the OS-level trust
// boundary of the subprocess: an MCP host spawns `authorizer mcp` as a child, and
// only that process can write to its stdin. Identity is the process-wide
// --mcp-bearer, so one process serves exactly one user. That is why it is a
// development transport and is deprecated for removal in 2.5.0.
//
// The earlier "stdio is the ONLY supported transport" constraint, and the
// TestServer_StdioOnly guard that enforced it, named their own exit condition:
// implement an auth interceptor for MCP first, then allow a network transport.
// That is what interceptors.MCPTokenResolver and the sole-authority guard are.
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
// whose proto annotation has `(authorizer.v1.mcp_tool).exposed = true`.
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
		log:           log,
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

// stampAuth attaches the caller's credential to the outgoing in-process gRPC
// call. This is the bridge that lets gRPC handlers see "who is calling".
//
// Over HTTP the credential is per REQUEST: reqHeader carries the Authorization
// header of the HTTP request that produced this tool call, so one server serves
// every caller under their own identity. Over stdio there is no HTTP request, so
// it falls back to the process-wide --mcp-bearer — one process, one user, which
// is exactly the limitation that makes stdio a development-only transport.
//
// ONLY the Authorization header crosses. Nothing else from the HTTP request is
// forwarded, and that is deliberate: transport.MetaFromGRPC reconstructs cookies
// and x-authorizer-url from gRPC metadata, so forwarding headers wholesale would
// let a browser session cookie — or a host header — reach the auth path of a
// surface whose whole security model is "the token's audience must name this MCP
// server". The interceptor refuses those credentials too (see
// interceptors.Auth's sole-authority guard), but the bridge should not be
// offering them in the first place.
//
// x-authorizer-url is likewise NOT forwarded. MCP requires --url, so
// parsers.GetHost short-circuits to the operator-configured value and a header
// could only disagree with it.
func (s *Server) stampAuth(ctx context.Context, reqHeader http.Header) context.Context {
	if auth := reqHeader.Get("Authorization"); auth != "" {
		return metadata.AppendToOutgoingContext(ctx, "authorization", auth)
	}
	if s.bearer != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+s.bearer)
	}
	if s.authorizerURL != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-authorizer-url", s.authorizerURL)
	}
	return ctx
}

// Handler serves MCP over Streamable HTTP.
//
// Stateless + JSONResponse, for two independent reasons. The main HTTP listener
// sets WriteTimeout: 60s, which would kill a long-lived SSE stream mid-flight;
// and a stateless server needs no sticky sessions, so an Authorizer deployment
// can scale horizontally without the MCP surface pinning a client to one
// replica. In this mode the SDK answers GET with 405 + Allow, which is
// spec-conformant — every exposed tool is request/response, so no server→client
// stream is needed.
//
// Authentication is NOT done here. It happens twice, on purpose: the route
// wrapper rejects a bad credential with a 401 so the client knows to refresh or
// start discovery, and the in-process gRPC interceptor resolves the identity
// that handlers actually run under. See the route registration in
// internal/server.
func (s *Server) Handler() http.Handler {
	return mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return s.mcpSrv },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)
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

		// Extra is nil on transports that carry no HTTP request (stdio), where
		// stampAuth falls back to the process-wide bearer.
		var reqHeader http.Header
		if req.Extra != nil {
			reqHeader = req.Extra.Header
		}

		respMsg := dynamicpb.NewMessage(b.OutputDescriptor)
		if err := s.gwConn.Invoke(s.stampAuth(ctx, reqHeader), b.FullMethod, reqMsg, respMsg); err != nil {
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

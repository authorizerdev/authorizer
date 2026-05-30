package integration_tests

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	authmcp "github.com/authorizerdev/authorizer/internal/mcp"
	"github.com/authorizerdev/authorizer/internal/service"
)

// TestMCPStubReturnsError exercises the "MCP tool exposed in proto but its
// underlying gRPC handler is still a stub" path. This is the current state
// of get_user, get_current_session, and list_my_permissions: they appear in
// tools/list (proven by TestMCPListAndCallGetMeta) and a call must surface
// the underlying codes.Unimplemented as a tool error rather than silently
// succeeding or panicking.
func TestMCPStubReturnsError(t *testing.T) {
	cfg := getTestConfig()
	cfg.ClientID = "test-client"
	log := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	svc, err := service.New(cfg, &service.Dependencies{Log: &log})
	require.NoError(t, err)
	grpcSrv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{Log: &log, Config: cfg, ServiceProvider: svc})
	require.NoError(t, err)
	mcpSrv, err := authmcp.New(&log, grpcSrv.GRPCServer(), "authorizer-test", "v0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cTransport, sTransport := mcp.NewInMemoryTransports()
	serverSession, err := mcpSrv.MCPServer().Connect(ctx, sTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	clientSession, err := client.Connect(ctx, cTransport, nil)
	require.NoError(t, err)
	defer clientSession.Close()

	// permissions is exposed via the proto annotation but its
	// AuthorizerService.Permissions handler is a stub returning codes.Unimplemented.
	// The MCP server must surface this as a CallToolResult{IsError:true}
	// (tool-level error) rather than a JSON-RPC protocol error — so the
	// LLM gets actionable text and can react / try a different tool.
	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "permissions",
		Arguments: map[string]any{},
	})
	require.NoError(t, err, "tool execution errors must NOT surface as protocol errors")
	require.NotNil(t, res)
	assert.True(t, res.IsError, "stubbed tool must return IsError=true")
	require.NotEmpty(t, res.Content)
	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok, "error content should be text")
	assert.Contains(t, text.Text, "Unimplemented",
		"the underlying gRPC Unimplemented code should be reflected in the MCP error text")
}

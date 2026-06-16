package integration_tests

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	authmcp "github.com/authorizerdev/authorizer/internal/mcp"
)

// TestMCPToolErrorSurfacesAsIsErrorResult verifies that when the underlying
// gRPC handler returns a non-OK status, the MCP server surfaces it as a
// CallToolResult{IsError:true} (tool-level error) rather than as a
// JSON-RPC protocol error. This is the MCP-spec way to give the LLM
// actionable text it can react to (vs aborting the whole exchange).
//
// We exercise this by calling `check_permissions`: with no FGA engine wired
// in the test config, the CheckPermissions handler fails closed at the
// service layer with "fine-grained authorization is not enabled"
// (service.ErrFgaNotEnabled). That gRPC error must reach the client as a
// tool-level error (IsError:true) carrying the message as text, not as a
// JSON-RPC protocol error — proving the fail-closed gate surfaces cleanly
// and auditably to the MCP host.
func TestMCPToolErrorSurfacesAsIsErrorResult(t *testing.T) {
	cfg := getTestConfig()
	cfg.ClientID = "test-client"
	ts := initTestSetup(t, cfg)
	bearer := testAccessToken(t, ts)
	authorizerURL := testAuthorizerHost(ts)

	grpcSrv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             ts.Logger,
		Config:          cfg,
		ServiceProvider: ts.ServiceProvider,
		TokenProvider:   ts.TokenProvider,
	})
	require.NoError(t, err)

	mcpSrv, err := authmcp.New(ts.Logger, grpcSrv.GRPCServer(), authmcp.Options{
		Name:          "authorizer-test",
		Version:       "v0",
		Bearer:        bearer,
		AuthorizerURL: authorizerURL,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cTransport, sTransport := mcp.NewInMemoryTransports()
	serverSession, err := mcpSrv.MCPServer().Connect(ctx, sTransport, nil)
	require.NoError(t, err)
	defer func() { _ = serverSession.Close() }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	clientSession, err := client.Connect(ctx, cTransport, nil)
	require.NoError(t, err)
	defer func() { _ = clientSession.Close() }()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "check_permissions",
		Arguments: map[string]any{
			"checks": []map[string]any{
				{"relation": "can_view", "object": "document:1"},
			},
		},
	})
	require.NoError(t, err, "tool execution errors must NOT surface as protocol errors")
	require.NotNil(t, res)
	assert.True(t, res.IsError, "check_permissions with FGA disabled must return IsError=true")
	require.NotEmpty(t, res.Content)
	txt, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok, "error content should be text")
	assert.Contains(t, txt.Text, "fine-grained authorization is not enabled",
		"fail-closed FGA error message must surface to the MCP host")
}

package integration_tests

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	authmcp "github.com/authorizerdev/authorizer/internal/mcp"
	"github.com/authorizerdev/authorizer/internal/service"
)

// TestMCPListAndCallGetMeta exercises the Phase 4 vertical slice end-to-end:
// boot a gRPC server, wrap it in the MCP server (which auto-discovers tools
// from proto annotations), connect a client via in-memory transports, then
// list_tools + call get_meta.
func TestMCPListAndCallGetMeta(t *testing.T) {
	cfg := getTestConfig()
	cfg.ClientID = "test-client"

	log := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()

	svc, err := service.New(cfg, &service.Dependencies{Log: &log})
	require.NoError(t, err)

	grpcSrv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             &log,
		Config:          cfg,
		ServiceProvider: svc,
	})
	require.NoError(t, err)

	mcpSrv, err := authmcp.New(&log, grpcSrv.GRPCServer(), "authorizer-test", "v0")
	require.NoError(t, err)

	// Wire client ↔ server via in-memory transports (no stdio).
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

	// tools/list — should include get_meta (the only proto-annotated MCP tool today).
	list, err := clientSession.ListTools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, list.Tools, "expected at least one MCP-exposed tool")
	var found bool
	for _, tool := range list.Tools {
		if tool.Name == "get_meta" {
			found = true
			break
		}
	}
	require.True(t, found, "expected get_meta tool to be exposed")

	// tools/call get_meta — should invoke MetaService.GetMeta and return JSON.
	call, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_meta",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.NotNil(t, call.StructuredContent)

	body, err := json.Marshal(call.StructuredContent)
	require.NoError(t, err)
	var got struct {
		ClientID string `json:"client_id"`
		Version  string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	require.Equal(t, "test-client", got.ClientID)
	require.NotEmpty(t, got.Version)
}

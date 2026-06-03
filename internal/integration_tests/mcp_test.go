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

// TestMCPListAndCallMeta exercises the vertical slice end-to-end on the
// consolidated single-service design: boot a gRPC server, wrap it in the
// MCP server (which auto-discovers tools from proto annotations), connect a
// client via in-memory transports, then list_tools + call meta.
func TestMCPListAndCallMeta(t *testing.T) {
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

	mcpSrv, err := authmcp.New(&log, grpcSrv.GRPCServer(), authmcp.Options{Name: "authorizer-test", Version: "v0"})
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

	// tools/list — should include the three proto-annotated MCP tools:
	// meta, profile, permissions. (Session was DROPPED from MCP exposure
	// in the security pass; its response carries credentials that
	// shouldn't land in an LLM transcript — audit finding C1.)
	list, err := clientSession.ListTools(ctx, nil)
	require.NoError(t, err)
	gotNames := map[string]bool{}
	for _, tool := range list.Tools {
		gotNames[tool.Name] = true
	}
	for _, want := range []string{"meta", "profile", "permissions"} {
		require.True(t, gotNames[want], "expected MCP tool %q to be exposed; got %v", want, gotNames)
	}
	require.False(t, gotNames["session"],
		"session tool MUST NOT be exposed via MCP (carries access_token/refresh_token/etc.)")

	// tools/call meta — should invoke AuthorizerService.Meta and return JSON
	// wrapped in the per-RPC MetaResponse shape.
	call, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "meta",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.NotNil(t, call.StructuredContent)

	body, err := json.Marshal(call.StructuredContent)
	require.NoError(t, err)
	var got struct {
		Meta struct {
			ClientID string `json:"client_id"`
			Version  string `json:"version"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	require.Equal(t, "test-client", got.Meta.ClientID)
	require.NotEmpty(t, got.Meta.Version)
}

package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	"github.com/authorizerdev/authorizer/internal/mcp"
	"github.com/authorizerdev/authorizer/internal/service"
)

// mcpCmd serves Authorizer's MCP surface over stdio. Designed to be wired
// into Claude Code or any other MCP host via:
//
//	claude mcp add authorizer -- /path/to/authorizer mcp --client-id=... \
//	  --database-type=sqlite --database-url=auth.db
//
// Which tools are exposed is declared at the proto layer via the
// `(authorizer.common.v1.mcp_tool).exposed` option; the MCP server discovers
// them at startup. Today: GetMeta. As more public ops migrate into
// internal/service and get the mcp_tool annotation, they appear automatically.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Serve Authorizer's MCP tool surface over stdio",
	Long: "Exposes a subset of Authorizer's gRPC methods (those marked " +
		"(authorizer.common.v1.mcp_tool).exposed=true in proto) as MCP " +
		"tools, suitable for use with Claude Code or any MCP-compatible host.",
	Run: runMCP,
}

func init() {
	RootCmd.AddCommand(mcpCmd)
}

func runMCP(_ *cobra.Command, _ []string) {
	// MCP stdio mode: stderr-only logging so it doesn't interleave with the
	// JSON-RPC framing on stdout.
	log := zerolog.New(os.Stderr).With().Timestamp().Logger()

	// For the GetMeta-only vertical slice we don't need storage / token /
	// memory store / events / email / sms. As more MCP-exposed tools come
	// online (Phase 4+ migrations of ListMyPermissions, GetCurrentSession,
	// GetUser(me)) wire them in following the same pattern as runRoot.
	svc, err := service.New(&rootArgs.config, &service.Dependencies{
		Log: &log,
		// nil-safe: methods that need these subsystems are not yet exposed
		// as MCP tools. Each panics-on-nil call moved here would be caught
		// by integration tests before reaching prod.
		AuditProvider:       audit.New(&audit.Dependencies{Log: &log}),
		EmailProvider:       nil,
		EventsProvider:      nil,
		MemoryStoreProvider: nil,
		SMSProvider:         nil,
		StorageProvider:     nil,
		TokenProvider:       nil,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create service provider")
	}

	grpcSrv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             &log,
		Config:          &rootArgs.config,
		ServiceProvider: svc,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create grpc server")
	}

	mcpSrv, err := mcp.New(&log, grpcSrv.GRPCServer(), "authorizer", constants.VERSION)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create mcp server")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		cancel()
	}()

	if err := mcpSrv.RunStdio(ctx); err != nil {
		log.Error().Err(err).Msg("mcp server exited")
		os.Exit(1)
	}
}

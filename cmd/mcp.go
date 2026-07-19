package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/authenticators/webauthn"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/events"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	"github.com/authorizerdev/authorizer/internal/mcp"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/sms"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/token"
)

// mcpArgs are the MCP-subcommand-only flags. The root command's flags
// (--database-type, --client-id, --jwt-secret, ...) are inherited by the
// subcommand automatically since they live on RootCmd.
var mcpArgs struct {
	// bearer is propagated as `Authorization: Bearer <bearer>` on every
	// outgoing gRPC call. Without it the MCP server runs anonymously —
	// fine for the `meta` tool (public) but identity-bearing tools
	// (`profile`, `check_permissions`, `list_permissions`) won't have a
	// caller to attribute to.
	bearer string
	// authorizerURL is the public URL of the Authorizer instance that
	// minted the bearer token; stamped as `x-authorizer-url` so JWT issuer
	// validation passes for identity-bearing tools.
	authorizerURL string
}

// mcpCmd serves Authorizer's MCP surface over stdio. Designed to be wired
// into Claude Code or any other MCP host via:
//
//	claude mcp add authorizer -- /path/to/authorizer mcp --client-id=... \
//	  --database-type=sqlite --database-url=auth.db --mcp-bearer=$TOKEN
//
// Which tools are exposed is declared at the proto layer via the
// `(authorizer.v1.mcp_tool).exposed` option; the MCP server discovers
// them at startup.
//
// Transport: STDIO ONLY. The MCP server has no auth/rate-limit interceptors
// of its own — the security model relies on the OS-level trust boundary of
// the subprocess. See internal/mcp/server.go's Server type comment.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Serve Authorizer's MCP tool surface over stdio",
	Long: "Exposes a subset of Authorizer's gRPC methods (those marked " +
		"(authorizer.v1.mcp_tool).exposed=true in proto) as MCP " +
		"tools, suitable for use with Claude Code or any MCP-compatible " +
		"host. Stdio is the only supported transport.",
	Run: runMCP,
}

func init() {
	mcpCmd.Flags().StringVar(&mcpArgs.bearer, "mcp-bearer", "",
		"Bearer token to attach to every outgoing gRPC call (carries the "+
			"user identity for tools like Profile / Permissions / Session). "+
			"When unset the MCP server runs anonymously; public tools (Meta) "+
			"still work but identity-bearing tools will fail authn.")
	mcpCmd.Flags().StringVar(&mcpArgs.authorizerURL, "mcp-authorizer-url", "",
		"Public URL of the Authorizer instance that issued --mcp-bearer "+
			"(e.g. https://auth.example.com). Required with --mcp-bearer: "+
			"JWT issuer validation compares the token's iss claim against "+
			"this value.")
	RootCmd.AddCommand(mcpCmd)
}

func runMCP(_ *cobra.Command, _ []string) {
	// MCP stdio mode: stderr-only logging so it doesn't interleave with the
	// JSON-RPC framing on stdout.
	log := zerolog.New(os.Stderr).With().Timestamp().Logger()

	// Wire all subsystems an MCP-exposed tool might need. As more ops
	// migrate into internal/service, this list stays the same — the
	// service-provider dependencies don't change per op, only the methods
	// on the provider do.
	storageProvider, err := storage.New(&rootArgs.config, &storage.Dependencies{Log: &log})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create storage provider")
	}
	memoryStoreProvider, err := memory_store.New(&rootArgs.config, &memory_store.Dependencies{
		Log:             &log,
		StorageProvider: storageProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create memory store provider")
	}
	tokenProvider, err := token.New(&rootArgs.config, &token.Dependencies{
		Log:                 &log,
		MemoryStoreProvider: memoryStoreProvider,
		StorageProvider:     storageProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create token provider")
	}
	emailProvider, err := email.New(&rootArgs.config, &email.Dependencies{
		Log:             &log,
		StorageProvider: storageProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create email provider")
	}
	smsProvider, err := sms.New(&rootArgs.config, &sms.Dependencies{Log: &log})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create sms provider")
	}
	auditProvider := audit.New(&audit.Dependencies{
		Log:             &log,
		StorageProvider: storageProvider,
	})
	eventsProvider, err := events.New(&rootArgs.config, &events.Dependencies{
		Log:             &log,
		StorageProvider: storageProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create events provider")
	}

	// Embedded OpenFGA engine, shared init with root.go. Nil (fail-closed)
	// when FGA is not configured or init fails — the permission tools fail
	// closed while the rest of the MCP surface serves.
	authzEngine, closeAuthzEngine := initAuthzEngine(&rootArgs.config, &log)
	defer closeAuthzEngine()

	// Authenticator provider — required by the service layer's TOTP/MFA
	// verification flows (verify_otp, login).
	authenticatorProvider, err := authenticators.New(&rootArgs.config, &authenticators.Dependencies{
		Log:                 &log,
		StorageProvider:     storageProvider,
		MemoryStoreProvider: memoryStoreProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create authenticator provider")
	}

	webAuthnProvider, err := webauthn.NewProvider(&webauthn.Dependencies{
		Log:                 &log,
		StorageProvider:     storageProvider,
		MemoryStoreProvider: memoryStoreProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create webauthn provider")
	}

	svc, err := service.New(&rootArgs.config, &service.Dependencies{
		Log:                   &log,
		AuditProvider:         auditProvider,
		AuthenticatorProvider: authenticatorProvider,
		AuthzEngine:           authzEngine,
		EmailProvider:         emailProvider,
		EventsProvider:        eventsProvider,
		MemoryStoreProvider:   memoryStoreProvider,
		SMSProvider:           smsProvider,
		StorageProvider:       storageProvider,
		TokenProvider:         tokenProvider,
		WebAuthnProvider:      webAuthnProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create service provider")
	}

	grpcSrv, err := grpcsrv.New(":0", &grpcsrv.Dependencies{
		Log:             &log,
		Config:          &rootArgs.config,
		ServiceProvider: svc,
		TokenProvider:   tokenProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create grpc server")
	}

	mcpSrv, err := mcp.New(&log, grpcSrv.GRPCServer(), mcp.Options{
		Name:          "authorizer",
		Version:       constants.VERSION,
		Bearer:        mcpArgs.bearer,
		AuthorizerURL: mcpArgs.authorizerURL,
	})
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

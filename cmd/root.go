package cmd

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	// "golang.org/x/sync/errgroup"

	"github.com/authorizerdev/authorizer/internal/models"
	"github.com/authorizerdev/authorizer/internal/server"
)

var (
	RootCmd = cobra.Command{
		Use: "authorizer",
		Run: runRoot,
	}
	rootArgs struct {
		server   server.Config
		models   models.Config
		logLevel string
	}
)

const (
	defaultHost        = "0.0.0.0"
	defaultHTTPPort    = 8080
	defaultMetricsPort = 8081
)

func init() {
	f := RootCmd.Flags()

	// Server flags
	f.StringVar(&rootArgs.server.Host, "host", defaultHost, "Host address to listen on")
	f.IntVar(&rootArgs.server.HTTPPort, "http-port", defaultHTTPPort, "Port to serve HTTP requests on")
	f.IntVar(&rootArgs.server.MetricsPort, "metrics-port", defaultMetricsPort, "Port to serve metrics requests on")

	// Logging flags
	f.StringVar(&rootArgs.logLevel, "log-level", "debug", "Log level to use")

	// Database flags
	f.StringVar(&rootArgs.models.DatabaseType, "database-type", "", "Type of database to use")
	f.StringVar(&rootArgs.models.DatabaseURL, "database-url", "", "URL of the database")

	// Deprecated flags
	f.MarkDeprecated("database_url", "use --database-url instead")
	f.MarkDeprecated("database_type", "use --database-type instead")
	f.MarkDeprecated("env_file", "use --env-file instead")
	f.MarkDeprecated("log_level", "use --log-level instead")
	f.MarkDeprecated("redis_url", "use --redis-url instead")
}

// Run the service
func runRoot(c *cobra.Command, args []string) {
	// Prepare logger
	ctx := context.Background()
	// Parse the log level
	zeroLogLevel, err := zerolog.ParseLevel(rootArgs.logLevel)
	if err != nil {
		// If the log level is invalid, set it to debug
		zeroLogLevel = zerolog.DebugLevel
	}
	// Create a new console writer
	consoleWriter := zerolog.NewConsoleWriter()
	consoleWriter.NoColor = true
	consoleWriter.TimeFormat = time.RFC3339
	consoleWriter.TimeLocation = time.UTC
	log := zerolog.New(consoleWriter).
		Level(zeroLogLevel).
		With().Timestamp().Logger()

	// Prepare server
	deps := server.Dependencies{
		Log: log,
	}
	// Create the server
	svr, err := server.New(rootArgs.server, deps)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create server")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return svr.Run(ctx)
	})

	// Setup signal handler to allow for graceful termination
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt)

	// Wait for interrupt or failure in errgroup.
	select {
	case <-sigCtx.Done():
		log.Info().Msg("Signal received, shutting down...")
		// Unregister signal handlers.
		// Next interrupt signal will kill us.
		cancel()
		stop()
	case <-ctx.Done():
		// Errgroup context canceled
	}

	// Wait for all routines to end
	if err := g.Wait(); err != nil {
		log.Fatal().Err(err).Msg("Application failed")
	}
	log.Info().Msg("Application terminated")
}

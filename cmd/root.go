package cmd

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/server"
)

var (
	RootCmd = cobra.Command{
		Use: "authorizer",
		Run: runRoot,
	}
	rootArgs struct {
		server   server.Config
		config   config.Config
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
	f.StringVar(&rootArgs.config.DatabaseType, "database-type", "", "Type of database to use")
	f.StringVar(&rootArgs.config.DatabaseURL, "database-url", "", "URL of the database")
	f.StringVar(&rootArgs.config.DatabaseName, "database-name", "", "Name of the database")
	f.StringVar(&rootArgs.config.DatabaseUsername, "database-username", "", "Username for the database")
	f.StringVar(&rootArgs.config.DatabasePassword, "database-password", "", "Password for the database")
	f.StringVar(&rootArgs.config.DatabaseHost, "database-host", "", "Host for the database")
	f.IntVar(&rootArgs.config.DatabasePort, "database-port", 0, "Port for the database")
	f.StringVar(&rootArgs.config.DatabaseCert, "database-cert", "", "Certificate for the database")
	f.StringVar(&rootArgs.config.DatabaseCACert, "database-ca-cert", "", "CA certificate for the database")
	f.StringVar(&rootArgs.config.DatabaseCertKey, "database-cert-key", "", "Certificate key for the database")
	f.StringVar(&rootArgs.config.CouchBaseBucket, "couchbase-bucket", "", "Bucket for the database")
	f.StringVar(&rootArgs.config.CouchBaseRamQuota, "couchbase-ram-quota", "", "RAM quota for the database")
	f.StringVar(&rootArgs.config.CouchBaseScope, "couchbase-scope", "", "Scope for the database")
	f.StringVar(&rootArgs.config.AWSRegion, "aws-region", "", "Region for the dynamodb database")
	f.StringVar(&rootArgs.config.AWSAccessKeyID, "aws-access-key-id", "", "Access key ID for the dynamodb database")
	f.StringVar(&rootArgs.config.AWSSecretAccessKey, "aws-secret-access-key", "", "Secret access key for the dynamodb database")

	// Memory store flags
	f.StringVar(&rootArgs.config.RedisURL, "redis-url", "", "URL of the redis server")

	// Email flags
	f.StringVar(&rootArgs.config.SMTPHost, "smtp-host", "", "Host for the SMTP server")
	f.IntVar(&rootArgs.config.SMTPPort, "smtp-port", 0, "Port for the SMTP server")
	f.StringVar(&rootArgs.config.SMTPUsername, "smtp-username", "", "Username for the SMTP server")
	f.StringVar(&rootArgs.config.SMTPPassword, "smtp-password", "", "Password for the SMTP server")
	f.StringVar(&rootArgs.config.SMTPSenderEmail, "smtp-sender-email", "", "Sender email for the SMTP server")
	f.StringVar(&rootArgs.config.SMTPSenderName, "smtp-sender-name", "", "Sender name for the SMTP server")
	f.StringVar(&rootArgs.config.SMTPLocalName, "smtp-local-name", "", "Local name for the SMTP server")
	f.BoolVar(&rootArgs.config.SkipTLSVerification, "skip-tls-verification", false, "Skip TLS verification for the SMTP server")

	// User flags
	f.StringVar(&rootArgs.config.DefaultRoles, "default-roles", "user", "Default user roles to assign")

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

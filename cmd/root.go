package cmd

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/events"
	"github.com/authorizerdev/authorizer/internal/http_handlers"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/server"
	"github.com/authorizerdev/authorizer/internal/sms"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/token"
)

var (
	RootCmd = cobra.Command{
		Use: "authorizer",
		Run: runRoot,
	}
	rootArgs struct {
		logLevel string
		config   config.Config
		server   server.Config
	}
)

func init() {
	f := RootCmd.Flags()

	// Server flags
	f.StringVar(&rootArgs.server.Host, "host", "0.0.0.0", "Host address to listen on")
	f.IntVar(&rootArgs.server.HTTPPort, "http-port", 8080, "Port to serve HTTP requests on")
	f.IntVar(&rootArgs.server.MetricsPort, "metrics-port", 8081, "Port to serve metrics requests on")

	// Logging flags
	f.StringVar(&rootArgs.logLevel, "log-level", "debug", "Log level to use")

	// Env
	f.StringVar(&rootArgs.config.Env, "env", "", "Environment of the authorizer instance")

	// Http routes
	f.BoolVar(&rootArgs.config.DisableLoginPage, "disable-login-page", false, "Disable login page")
	f.BoolVar(&rootArgs.config.DisablePlayground, "disable-playground", false, "Disable playground")

	// Organization flags
	f.StringVar(&rootArgs.config.OrganizationLogo, "organization-logo", "", "Logo of the organization")
	f.StringVar(&rootArgs.config.OrganizationName, "organization-name", "", "Name of the organization")

	// OAuth flags
	f.StringVar(&rootArgs.config.ClientID, "client-id", "", "Client ID for the OAuth")
	f.StringVar(&rootArgs.config.ClientSecret, "client-secret", "", "Client secret for the OAuth")
	f.StringVar(&rootArgs.config.DefaultAuthorizeResponseMode, "default-authorize-response-mode", constants.ResponseModeQuery, "Default response mode for the authorize endpoint")
	f.StringVar(&rootArgs.config.DefaultAuthorizeResponseType, "default-authorize-response-type", constants.ResponseTypeToken, "Default response type for the authorize endpoint")

	// Admin flags
	f.StringVar(&rootArgs.config.AdminSecret, "admin-secret", "password", "Secret for the admin")

	// Allowed origins
	f.StringSliceVar(&rootArgs.config.AllowedOrigins, "allowed-origins", []string{"*"}, "Allowed origins")

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

	// Auth flags
	f.StringSliceVar(&rootArgs.config.DefaultRoles, "default-roles", []string{"user"}, "Default user roles to assign")
	f.StringSliceVar(&rootArgs.config.Roles, "roles", []string{"user"}, "Roles to assign")
	f.StringSliceVar(&rootArgs.config.ProtectedRoles, "protected-roles", []string{}, "Roles that cannot be deleted")
	f.BoolVar(&rootArgs.config.DisableStrongPassword, "disable-strong-password", false, "Disable strong password requirement")
	f.BoolVar(&rootArgs.config.DisableTOTPLogin, "disable-totp-login", false, "Disable TOTP login")
	f.BoolVar(&rootArgs.config.DisableBasicAuthentication, "disable-basic-authentication", false, "Disable basic authentication")
	f.BoolVar(&rootArgs.config.DisableEmailVerification, "disable-email-verification", false, "Disable email verification")
	f.BoolVar(&rootArgs.config.DisableMobileBasicAuthentication, "disable-mobile-basic-authentication", false, "Disable mobile basic authentication")
	f.BoolVar(&rootArgs.config.DisablePhoneVerification, "disable-phone-verification", false, "Disable phone verification")
	f.BoolVar(&rootArgs.config.DisableMagicLinkLogin, "disable-magic-link-login", false, "Disable magic link login")
	f.BoolVar(&rootArgs.config.EnforceMFA, "enforce-mfa", false, "Enforce MFA for all users")
	f.BoolVar(&rootArgs.config.DisableMFA, "disable-mfa", false, "Disable MFA for all users")
	f.BoolVar(&rootArgs.config.DisableEmailOTP, "disable-email-otp", false, "Disable email OTP")
	f.BoolVar(&rootArgs.config.DisableSMSOTP, "disable-sms-otp", false, "Disable SMS OTP")
	f.BoolVar(&rootArgs.config.DisableSignup, "disable-signup", false, "Disable signup")

	// JWT flags
	f.StringVar(&rootArgs.config.JWTType, "jwt-type", "", "Type of JWT to use")
	f.StringVar(&rootArgs.config.JWTSecret, "jwt-secret", "", "Secret for the JWT")
	f.StringVar(&rootArgs.config.JWTPrivateKey, "jwt-private-key", "", "Private key for the JWT")
	f.StringVar(&rootArgs.config.JWTPublicKey, "jwt-public-key", "", "Public key for the JWT")
	f.StringVar(&rootArgs.config.JWTRoleClaim, "jwt-role-claim", "role", "Role claim for the JWT")
	f.StringVar(&rootArgs.config.CustomAccessTokenScript, "custom-access-token-script", "", "Custom access token script")

	// Twilio flags
	f.StringVar(&rootArgs.config.TwilioAccountSID, "twilio-account-sid", "", "Account SID for Twilio")
	f.StringVar(&rootArgs.config.TwilioAPIKey, "twilio-api-key", "", "API key for Twilio")
	f.StringVar(&rootArgs.config.TwilioAPISecret, "twilio-api-secret", "", "API secret for Twilio")
	f.StringVar(&rootArgs.config.TwilioSender, "twilio-sender", "", "Sender for Twilio")

	// Oauth provider flags
	f.StringVar(&rootArgs.config.GoogleClientID, "google-client-id", "", "Client ID for Google")
	f.StringVar(&rootArgs.config.GoogleClientSecret, "google-client-secret", "", "Client secret for Google")
	f.StringVar(&rootArgs.config.GithubClientID, "github-client-id", "", "Client ID for Github")
	f.StringVar(&rootArgs.config.GithubClientSecret, "github-client-secret", "", "Client secret for Github")
	f.StringVar(&rootArgs.config.FacebookClientID, "facebook-client-id", "", "Client ID for Facebook")
	f.StringVar(&rootArgs.config.FacebookClientSecret, "facebook-client-secret", "", "Client secret for Facebook")
	f.StringVar(&rootArgs.config.MicrosoftClientID, "microsoft-client-id", "", "Client ID for Microsoft")
	f.StringVar(&rootArgs.config.MicrosoftClientSecret, "microsoft-client-secret", "", "Client secret for Microsoft")
	f.StringVar(&rootArgs.config.MicrosoftTenantID, "microsoft-tenant-id", "", "Tenant ID for Microsoft")
	f.StringVar(&rootArgs.config.TwitchClientID, "twitch-client-id", "", "Client ID for Twitch")
	f.StringVar(&rootArgs.config.TwitchClientSecret, "twitch-client-secret", "", "Client secret for Twitch")
	f.StringVar(&rootArgs.config.LinkedinClientID, "linkedin-client-id", "", "Client ID for Linkedin")
	f.StringVar(&rootArgs.config.LinkedinClientSecret, "linkedin-client-secret", "", "Client secret for Linkedin")
	f.StringVar(&rootArgs.config.AppleClientID, "apple-client-id", "", "Client ID for Apple")
	f.StringVar(&rootArgs.config.AppleClientSecret, "apple-client-secret", "", "Client secret for Apple")
	f.StringVar(&rootArgs.config.DiscordClientID, "discord-client-id", "", "Client ID for Discord")
	f.StringVar(&rootArgs.config.DiscordClientSecret, "discord-client-secret", "", "Client secret for Discord")
	f.StringVar(&rootArgs.config.TwitterClientID, "twitter-client-id", "", "Client ID for Twitter")
	f.StringVar(&rootArgs.config.TwitterClientSecret, "twitter-client-secret", "", "Client secret for Twitter")
	f.StringVar(&rootArgs.config.RoboloxClientID, "roblox-client-id", "", "Client ID for Roblox")
	f.StringVar(&rootArgs.config.RoboloxClientSecret, "roblox-client-secret", "", "Client secret for Roblox")

	// URLs
	f.StringVar(&rootArgs.config.ResetPasswordURL, "reset-password-url", "", "URL for reset password")

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
	// consoleWriter := zerolog.New(os.Stdout)
	// consoleWriter.NoColor = true
	// consoleWriter.TimeFormat = time.RFC3339
	// consoleWriter.TimeLocation = time.UTC
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}
	log := zerolog.New(os.Stdout).
		Level(zeroLogLevel).
		With().Timestamp().Logger()

	// Storage provider
	storageProvider, err := storage.New(&rootArgs.config, &storage.Dependencies{
		Log: &log,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create storage provider")
	}

	// Authenticator provider
	authenticatorProvider, err := authenticators.New(&rootArgs.config, &authenticators.Dependencies{
		Log:             &log,
		StorageProvider: storageProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create authenticator provider")
	}

	// Email provider
	emailProvider, err := email.New(&rootArgs.config, &email.Dependencies{
		Log:             &log,
		StorageProvider: storageProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create email provider")
	}

	// Events provider
	eventsProvider, err := events.New(&rootArgs.config, &events.Dependencies{
		Log:             &log,
		StorageProvider: storageProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create events provider")
	}

	// Memory store provider
	memoryStoreProvider, err := memory_store.New(&rootArgs.config, &memory_store.Dependencies{
		Log: &log,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create memory store provider")
	}

	// SMS provider
	smsProvider, err := sms.New(&rootArgs.config, &sms.Dependencies{
		Log: &log,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create sms provider")
	}

	// Token provider
	tokenProvider, err := token.New(&rootArgs.config, &token.Dependencies{
		Log:                 &log,
		MemoryStoreProvider: memoryStoreProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create token provider")
	}

	// Ensure client ID and secret are set for authorizer instance
	if strings.TrimSpace(rootArgs.config.ClientID) == "" {
		log.Fatal().Msg("client ID missing in rootArgs")
	}

	if strings.TrimSpace(rootArgs.config.ClientSecret) == "" {
		log.Fatal().Msg("client secret missing in rootArgs")
	}

	httpProvider, err := http_handlers.New(&rootArgs.config, &http_handlers.Dependencies{
		Log:                   &log,
		AuthenticatorProvider: authenticatorProvider,
		EmailProvider:         emailProvider,
		EventsProvider:        eventsProvider,
		MemoryStoreProvider:   memoryStoreProvider,
		SMSProvider:           smsProvider,
		StorageProvider:       storageProvider,
		TokenProvider:         tokenProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create http provider")
	}
	// Prepare server
	deps := &server.Dependencies{
		Log:          &log,
		HTTPProvider: httpProvider,
	}
	// Create the server
	svr, err := server.New(&rootArgs.server, deps)
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

package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/authenticators/webauthn"
	"github.com/authorizerdev/authorizer/internal/clientmetadata"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/events"
	"github.com/authorizerdev/authorizer/internal/grpcsrv"
	"github.com/authorizerdev/authorizer/internal/grpcsrv/interceptors"
	"github.com/authorizerdev/authorizer/internal/http_handlers"
	scimhttp "github.com/authorizerdev/authorizer/internal/http_handlers/scim"
	"github.com/authorizerdev/authorizer/internal/mcp"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/oauth"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/rate_limit"
	"github.com/authorizerdev/authorizer/internal/server"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/service/scim"
	"github.com/authorizerdev/authorizer/internal/sms"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/token"
)

// Default values for flags (single source of truth for init and applyFlagDefaults).
var (
	defaultHost             = "0.0.0.0"
	defaultMetricsHost      = "127.0.0.1"
	defaultLogLevel         = "debug"
	defaultHTTPPort         = 8080
	defaultMetricsPort      = 8081
	defaultOrganizationLogo = "https://authorizer.dev/images/logo.png"
	defaultOrganizationName = "Authorizer"
	// defaultAdminSecret intentionally REMOVED. Admin secret must be supplied
	// explicitly via --admin-secret. The startup check in runRoot rejects
	// only the empty value; the strength of the supplied secret is the
	// operator's responsibility.
	defaultJWTRoleClaim      = "role"
	defaultMicrosoftTenantID = "common"
	defaultAllowedOrigins    = []string{"*"}
	defaultRoles             = []string{"user"}
	defaultGoogleScopes      = []string{"openid", "profile", "email"}
	defaultGithubScopes      = []string{"read:user", "user:email"}
	defaultFacebookScopes    = []string{"public_profile", "email"}
	defaultMicrosoftScopes   = []string{"openid", "profile", "email"}
	defaultTwitchScopes      = []string{"openid", "user:read:email"}
	// LinkedIn's current product is "Sign In with LinkedIn using OpenID
	// Connect"; the legacy r_liteprofile/r_emailaddress scopes are not
	// provisioned for apps onboarded to it.
	defaultLinkedinScopes = []string{"openid", "profile", "email"}
	defaultAppleScopes    = []string{"email", "name"}
	defaultDiscordScopes  = []string{"identify", "email"}
	defaultTwitterScopes  = []string{"tweet.read", "users.read"}
	defaultRobloxScopes   = []string{"openid", "profile"}
	// Default RPS cap per IP; raised from 10 to reduce false positives on busy UIs.
	defaultRateLimitRPS   = 30
	defaultRateLimitBurst = 20
)

var (
	RootCmd = cobra.Command{
		Use: "authorizer",
		// Derive runtime config (service availability, MFA defaults) before any
		// subcommand runs so `authorizer` and `authorizer mcp` stay consistent.
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			rootArgs.config.Finalize()
			// Fail closed rather than silently encrypting TOTP seeds with a
			// publicly computable key derived from an empty secret.
			if err := rootArgs.config.ValidateEncryptionKey(); err != nil {
				return err
			}
			// A mistyped SameSite silently becomes lax — see the doc comment.
			return rootArgs.config.ValidateAppCookieSameSite()
		},
		Run: runRoot,
	}
	rootArgs struct {
		logLevel string
		config   config.Config
		server   server.Config
	}
)

func init() {
	// Persistent so subcommands (`authorizer mcp`) inherit the full server
	// flag surface (--database-type, --client-id, --fga-store, ...) and
	// share the same rootArgs storage.
	f := RootCmd.PersistentFlags()

	// Server flags
	f.StringVar(&rootArgs.server.Host, "host", defaultHost, "Host address to listen on")
	f.IntVar(&rootArgs.server.HTTPPort, "http-port", defaultHTTPPort, "Port to serve HTTP requests on")
	f.IntVar(&rootArgs.server.MetricsPort, "metrics-port", defaultMetricsPort, "Port for the dedicated /metrics listener (must differ from --http-port)")
	f.StringVar(&rootArgs.server.MetricsHost, "metrics-host", defaultMetricsHost, "Bind address for the dedicated /metrics listener (default loopback; use 0.0.0.0 when Prometheus scrapes from another host/pod)")

	// Logging flags
	f.StringVar(&rootArgs.logLevel, "log-level", defaultLogLevel, "Log level to use")

	// Env
	f.StringVar(&rootArgs.config.Env, "env", "", "Environment of the authorizer instance")

	// Http routes
	f.BoolVar(&rootArgs.config.EnableLoginPage, "enable-login-page", true, "Enable login page")
	f.BoolVar(&rootArgs.config.EnableOrgDiscovery, "enable-org-discovery", false, "Enable public organization (home-realm) discovery endpoint and the /app email-first SSO routing step (opt-in; off keeps the login page unchanged)")
	f.BoolVar(&rootArgs.config.EnablePlayground, "enable-playground", true, "Enable playground")
	f.BoolVar(&rootArgs.config.EnableGraphQLIntrospection, "enable-graphql-introspection", true, "Enable GraphQL introspection for the /graphql endpoint")
	f.BoolVar(&rootArgs.config.EnableHSTS, "enable-hsts", false, "Enable Strict-Transport-Security response header (only enable behind TLS)")
	f.BoolVar(&rootArgs.config.DisableCSP, "disable-csp", false, "Disable the default Content-Security-Policy response header")
	f.IntVar(&rootArgs.config.GraphQLMaxComplexity, "graphql-max-complexity", 300, "Maximum total complexity score for a single GraphQL operation")
	f.IntVar(&rootArgs.config.GraphQLMaxDepth, "graphql-max-depth", 15, "Maximum nesting depth of a GraphQL selection set")
	f.IntVar(&rootArgs.config.GraphQLMaxAliases, "graphql-max-aliases", 30, "Maximum total number of aliased fields per GraphQL operation")
	f.Int64Var(&rootArgs.config.GraphQLMaxBodyBytes, "graphql-max-body-bytes", 1<<20, "Maximum allowed GraphQL request body size in bytes (default 1MB)")

	// gRPC server flags. Port 9091 avoids collision with the metrics
	// listener which defaults to 8081 (and with the HTTP listener on 8080).
	f.IntVar(&rootArgs.config.GRPCPort, "grpc-port", 9091, "Port the gRPC server listens on")
	f.BoolVar(&rootArgs.config.EnableGRPCReflection, "enable-grpc-reflection", true, "Enable the gRPC server-reflection service")
	f.StringVar(&rootArgs.config.GRPCTLSCert, "grpc-tls-cert", "", "Path to the TLS certificate for the gRPC server")
	f.StringVar(&rootArgs.config.GRPCTLSKey, "grpc-tls-key", "", "Path to the TLS private key for the gRPC server")
	f.BoolVar(&rootArgs.config.GRPCInsecure, "grpc-insecure", false, "Allow the gRPC server to run without TLS (dev only)")

	// MCP transport. Served at POST /mcp on the main HTTP listener (not its own
	// port): it is plain HTTP that must be publicly reachable on the same origin
	// as the OAuth metadata clients discover it through, and mounting it on the
	// main router gives it the existing CORS, security-header, rate-limit and
	// logging middleware.
	f.BoolVar(&rootArgs.config.EnableClientIDMetadataDocument, "enable-client-id-metadata-document", false,
		"Accept HTTPS-URL client_ids that resolve to a Client ID Metadata Document (CIMD), so clients "+
			"with no prior relationship can authenticate — required for the OAuth path to the MCP surface. "+
			"Clients registered this way are self-asserted, so a consent screen is shown for them. Off by default")
	f.StringSliceVar(&rootArgs.config.ClientIDMetadataAllowedDomains, "client-id-metadata-allowed-domains", nil,
		"Restrict which hosts may serve a client metadata document (e.g. claude.ai). Empty accepts any HTTPS host")
	f.BoolVar(&rootArgs.config.EnableDynamicClientRegistration, "enable-dynamic-client-registration", false,
		"Serve the RFC 7591 dynamic client registration endpoint at POST <url>/oauth/register, for MCP "+
			"clients that cannot use CIMD. This is an UNAUTHENTICATED write endpoint: prefer "+
			"--enable-client-id-metadata-document where the client supports it. Clients registered this "+
			"way are self-asserted, so a consent screen is shown for them. Off by default")

	f.BoolVar(&rootArgs.config.MCPEnabled, "mcp-enabled", false,
		"Serve the MCP tool surface over HTTP at POST <url>/mcp as an OAuth 2.1 resource server. "+
			"Requires --url: tokens are accepted only when their audience equals <url>/mcp, and that "+
			"comparison must not depend on a request header. Off by default — it is a new "+
			"internet-facing authenticated surface")

	// Organization flags
	f.StringVar(&rootArgs.config.OrganizationLogo, "organization-logo", defaultOrganizationLogo, "Logo of the organization")
	f.StringVar(&rootArgs.config.OrganizationName, "organization-name", defaultOrganizationName, "Name of the organization")

	// OAuth flags
	f.StringVar(&rootArgs.config.ClientID, "client-id", "", "Client ID for the OAuth")
	f.StringVar(&rootArgs.config.ClientSecret, "client-secret", "", "Client secret for the OAuth")
	f.StringVar(&rootArgs.config.DefaultAuthorizeResponseMode, "default-authorize-response-mode", constants.ResponseModeQuery, "Default response mode for the authorize endpoint")
	f.StringVar(&rootArgs.config.DefaultAuthorizeResponseType, "default-authorize-response-type", constants.ResponseTypeToken, "Default response type for the authorize endpoint")
	f.BoolVar(&rootArgs.config.OAuth21Strict, "oauth2-1-strict", false, "Enforce OAuth 2.1 restrictions: reject the implicit/hybrid-with-token response types (token, id_token token) and PKCE plain (require S256). Breaking; opt-in")

	// Admin flags
	f.StringVar(&rootArgs.config.AdminSecret, "admin-secret", "", "Secret for the admin (REQUIRED, must not be empty)")
	f.Int64Var(&rootArgs.config.RefreshTokenExpiresIn, "refresh-token-expires-in", 60*60*24*30, "Refresh token lifetime in seconds (default: 30 days = 2592000)")

	// Allowed origins
	f.StringSliceVar(&rootArgs.config.AllowedOrigins, "allowed-origins", defaultAllowedOrigins, "Allowed origins")
	f.StringSliceVar(&rootArgs.config.RedirectURIs, "redirect-uris", nil, "Exact redirect URIs allowed for this deployment's own client (--client-id), comma-separated. When set, redirect_uri is matched EXACTLY against this list per OIDC Core §3.1.2.1 instead of falling back to --allowed-origins, which compares origins and so accepts any path under an allowed host. List every redirect URI your apps use: it applies to every flow carrying this client_id. Unset keeps the origin-based fallback")

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
	f.BoolVar(&rootArgs.config.SMTPSkipTLSVerification, "smtp-skip-tls-verification", false, "Skip TLS verification for the SMTP server")

	// Auth flags
	f.StringSliceVar(&rootArgs.config.DefaultRoles, "default-roles", defaultRoles, "Default user roles to assign")
	f.StringSliceVar(&rootArgs.config.Roles, "roles", defaultRoles, "Roles to assign")
	f.StringSliceVar(&rootArgs.config.ProtectedRoles, "protected-roles", []string{}, "Roles that cannot be deleted")
	f.BoolVar(&rootArgs.config.EnableStrongPassword, "enable-strong-password", true, "Enable strong password requirement")
	f.BoolVar(&rootArgs.config.EnableBasicAuthentication, "enable-basic-authentication", true, "Enable basic authentication")
	f.BoolVar(&rootArgs.config.EnableEmailVerification, "enable-email-verification", false, "Enable email verification")
	f.BoolVar(&rootArgs.config.EnableMobileBasicAuthentication, "enable-mobile-basic-authentication", true, "Enable mobile basic authentication")
	f.BoolVar(&rootArgs.config.EnablePhoneVerification, "enable-phone-verification", false, "Enable phone verification")
	f.BoolVar(&rootArgs.config.EnableMagicLinkLogin, "enable-magic-link-login", false, "Enable magic link login")
	// MFA is optional by default; set --enforce-mfa to require it. MFA methods
	// are enabled by default and opted out via --disable-* (email/SMS OTP only
	// take effect when their provider is configured). Config.Finalize() derives
	// the effective Enable* / EnableMFA values from these flags.
	f.BoolVar(&rootArgs.config.EnforceMFA, "enforce-mfa", false, "Enforce MFA for all users")
	f.BoolVar(&rootArgs.config.DisableTOTPLogin, "disable-totp-login", false, "Disable TOTP-based MFA (enabled by default)")
	f.BoolVar(&rootArgs.config.DisableWebauthnMFA, "disable-webauthn-mfa", false, "Disable WebAuthn/passkey as an MFA factor (enabled by default); does not affect WebAuthn/passkey as a primary login method")
	f.BoolVar(&rootArgs.config.DisableEmailOTP, "disable-email-otp", false, "Disable email OTP MFA (enabled by default when email service is configured)")
	f.BoolVar(&rootArgs.config.DisableSMSOTP, "disable-sms-otp", false, "Disable SMS OTP MFA (enabled by default when SMS service is configured)")
	f.BoolVar(&rootArgs.config.DisableMFA, "disable-mfa", false, "Globally disable MFA (TOTP/email/SMS OTP), overriding the per-method flags; does not affect WebAuthn/passkey as a primary login method")
	f.BoolVar(&rootArgs.config.EnableSignup, "enable-signup", true, "Enable signup")

	// Cookies flags
	f.BoolVar(&rootArgs.config.AppCookieSecure, "app-cookie-secure", true, "Application secure cookie flag")
	// Default "none" is deliberate and audit-reviewed, not an oversight: the
	// product targets an auth server on a subdomain serving apps on other
	// sites, and Lax withholds the session cookie on exactly those cross-site
	// requests. Same position Auth0 takes. See cookie.BuildSessionCookies for
	// the full reasoning before changing it.
	f.StringVar(&rootArgs.config.AppCookieSameSite, "app-cookie-same-site", "none", "SameSite attribute for session cookies (lax, strict, none). Default none supports apps on other domains; set lax if every app shares this host")
	f.BoolVar(&rootArgs.config.AdminCookieSecure, "admin-cookie-secure", true, "Admin secure cookie flag")
	f.BoolVar(&rootArgs.config.DisableAdminHeaderAuth, "disable-admin-header-auth", false, "Disable admin authentication via X-Authorizer-Admin-Secret header")

	// Rate limiting flags
	f.IntVar(&rootArgs.config.RateLimitRPS, "rate-limit-rps", defaultRateLimitRPS, "Maximum requests per second per IP for rate limiting")
	f.IntVar(&rootArgs.config.RateLimitBurst, "rate-limit-burst", defaultRateLimitBurst, "Maximum burst size per IP for rate limiting")
	f.BoolVar(&rootArgs.config.RateLimitFailClosed, "rate-limit-fail-closed", false, "On rate-limit backend errors, reject with 503 instead of allowing the request")
	f.StringSliceVar(&rootArgs.config.TrustedProxies, "trusted-proxies", nil, "Comma-separated CIDRs of trusted reverse proxies. When set, gin uses X-Forwarded-For from these networks. Empty (default) trusts no proxies and uses RemoteAddr.")

	// JWT flags
	f.StringVar(&rootArgs.config.JWTType, "jwt-type", "", "Type of JWT to use")
	f.StringVar(&rootArgs.config.JWTSecret, "jwt-secret", "", "Secret for the JWT")
	f.StringVar(&rootArgs.config.EncryptionKey, "encryption-key", "", "Key used to encrypt secrets at rest (TOTP seeds). Defaults to --jwt-secret for backwards compatibility; set a distinct value before rotating --jwt-secret, otherwise rotation locks out every enrolled TOTP user")
	f.StringVar(&rootArgs.config.JWTPrivateKey, "jwt-private-key", "", "Private key for the JWT")
	f.StringVar(&rootArgs.config.JWTPublicKey, "jwt-public-key", "", "Public key for the JWT")
	// JWT secondary key flags (for manual key rotation)
	f.StringVar(&rootArgs.config.JWTSecondaryType, "jwt-secondary-type", "", "Algorithm of the optional secondary JWT key used for manual rotation. When set, JWKS publishes both keys and token validation accepts either. New tokens are always signed with the primary (--jwt-type) key.")
	f.StringVar(&rootArgs.config.JWTSecondarySecret, "jwt-secondary-secret", "", "Secret for the secondary JWT key (HMAC only; never exposed via JWKS)")
	f.StringVar(&rootArgs.config.JWTSecondaryPrivateKey, "jwt-secondary-private-key", "", "Private key for the secondary JWT key. Currently unused — verification only uses the public key; kept for symmetry with --jwt-private-key and for future primary/secondary swap automation.")
	f.StringVar(&rootArgs.config.JWTSecondaryPublicKey, "jwt-secondary-public-key", "", "Public key for the secondary JWT key. Used to verify tokens signed with the secondary key during rotation.")
	f.StringVar(&rootArgs.config.JWTRoleClaim, "jwt-role-claim", defaultJWTRoleClaim, "Role claim for the JWT")
	f.StringVar(&rootArgs.config.CustomAccessTokenScript, "custom-access-token-script", "", "Custom access token script")

	// Twilio flags
	f.StringVar(&rootArgs.config.TwilioAccountSID, "twilio-account-sid", "", "Account SID for Twilio")
	f.StringVar(&rootArgs.config.TwilioAPIKey, "twilio-api-key", "", "API key for Twilio")
	f.StringVar(&rootArgs.config.TwilioAPISecret, "twilio-api-secret", "", "API secret for Twilio")
	f.StringVar(&rootArgs.config.TwilioSender, "twilio-sender", "", "Sender for Twilio")

	// Oauth provider flags
	f.StringVar(&rootArgs.config.GoogleClientID, "google-client-id", "", "Client ID for Google")
	f.StringVar(&rootArgs.config.GoogleClientSecret, "google-client-secret", "", "Client secret for Google")
	f.StringSliceVar(&rootArgs.config.GoogleScopes, "google-scopes", defaultGoogleScopes, "Scopes for Google")
	f.StringVar(&rootArgs.config.GithubClientID, "github-client-id", "", "Client ID for Github")
	f.StringVar(&rootArgs.config.GithubClientSecret, "github-client-secret", "", "Client secret for Github")
	f.StringSliceVar(&rootArgs.config.GithubScopes, "github-scopes", defaultGithubScopes, "Scopes for Github")
	f.StringVar(&rootArgs.config.FacebookClientID, "facebook-client-id", "", "Client ID for Facebook")
	f.StringVar(&rootArgs.config.FacebookClientSecret, "facebook-client-secret", "", "Client secret for Facebook")
	f.StringSliceVar(&rootArgs.config.FacebookScopes, "facebook-scopes", defaultFacebookScopes, "Scopes for Facebook")
	f.StringVar(&rootArgs.config.MicrosoftClientID, "microsoft-client-id", "", "Client ID for Microsoft")
	f.StringVar(&rootArgs.config.MicrosoftClientSecret, "microsoft-client-secret", "", "Client secret for Microsoft")
	f.StringVar(&rootArgs.config.MicrosoftTenantID, "microsoft-tenant-id", defaultMicrosoftTenantID, "Tenant ID for Microsoft")
	f.StringSliceVar(&rootArgs.config.MicrosoftScopes, "microsoft-scopes", defaultMicrosoftScopes, "Scopes for Microsoft")
	f.StringSliceVar(&rootArgs.config.MicrosoftAllowedTenants, "microsoft-allowed-tenants", nil, "Entra tenant IDs allowed to sign in when --microsoft-tenant-id is a multi-tenant alias (common/organizations/consumers). Empty allows any tenant, but an untrusted tenant's email will not link to an existing account")
	f.BoolVar(&rootArgs.config.FgaAllowUnconstrainedAgents, "fga-allow-unconstrained-agents", false, "When a delegated (agent-acting-for-user) FGA check runs against an authorization model with no `type agent`, authorize as the delegating user alone instead of denying. Discards the agent half of the permission intersection; add `type agent` to your model instead")
	f.BoolVar(&rootArgs.config.OAuthAllowUnverifiedProviderEmail, "oauth-allow-unverified-provider-email", false, "Compatibility escape hatch: let a social login whose provider did not attest the email address sign up or return to an account that same provider already owns. It still cannot cross into an account another credential owns. Prefer pinning --microsoft-tenant-id or enabling the xms_edov claim; see docs/email-verification-contract.md")
	f.StringVar(&rootArgs.config.TwitchClientID, "twitch-client-id", "", "Client ID for Twitch")
	f.StringVar(&rootArgs.config.TwitchClientSecret, "twitch-client-secret", "", "Client secret for Twitch")
	f.StringSliceVar(&rootArgs.config.TwitchScopes, "twitch-scopes", defaultTwitchScopes, "Scopes for Twitch")
	f.StringVar(&rootArgs.config.LinkedinClientID, "linkedin-client-id", "", "Client ID for Linkedin")
	f.StringVar(&rootArgs.config.LinkedinClientSecret, "linkedin-client-secret", "", "Client secret for Linkedin")
	f.StringSliceVar(&rootArgs.config.LinkedinScopes, "linkedin-scopes", defaultLinkedinScopes, "Scopes for Linkedin")
	f.StringVar(&rootArgs.config.AppleClientID, "apple-client-id", "", "Client ID for Apple")
	f.StringVar(&rootArgs.config.AppleClientSecret, "apple-client-secret", "", "Client secret for Apple")
	f.StringSliceVar(&rootArgs.config.AppleScopes, "apple-scopes", defaultAppleScopes, "Scopes for Apple")
	f.StringVar(&rootArgs.config.DiscordClientID, "discord-client-id", "", "Client ID for Discord")
	f.StringVar(&rootArgs.config.DiscordClientSecret, "discord-client-secret", "", "Client secret for Discord")
	f.StringSliceVar(&rootArgs.config.DiscordScopes, "discord-scopes", defaultDiscordScopes, "Scopes for Discord")
	f.StringVar(&rootArgs.config.TwitterClientID, "twitter-client-id", "", "Client ID for Twitter")
	f.StringVar(&rootArgs.config.TwitterClientSecret, "twitter-client-secret", "", "Client secret for Twitter")
	f.StringSliceVar(&rootArgs.config.TwitterScopes, "twitter-scopes", defaultTwitterScopes, "Scopes for Twitter")
	f.StringVar(&rootArgs.config.RobloxClientID, "roblox-client-id", "", "Client ID for Roblox")
	f.StringVar(&rootArgs.config.RobloxClientSecret, "roblox-client-secret", "", "Client secret for Roblox")
	f.StringSliceVar(&rootArgs.config.RobloxScopes, "roblox-scopes", defaultRobloxScopes, "Scopes for Roblox")

	// URLs
	f.StringVar(&rootArgs.config.AuthorizerURL, "url", "", "Canonical/trusted base URL of this Authorizer instance (e.g. https://auth.example.com). When set, it is the ONLY source used to build verification/reset/magic-link email URLs, the JWT iss claim, and OIDC discovery URLs; all request headers (X-Authorizer-URL, X-Forwarded-Host, Host) are ignored. REQUIRED: the server refuses to start without it, because deriving its own host from request headers exposes host-header-injection account takeover (CWE-640). This is not --allowed-origins: --url is this server's own address, --allowed-origins is the apps it may redirect to")
	f.StringVar(&rootArgs.config.ResetPasswordURL, "reset-password-url", "", "URL for reset password")

	// Back-channel logout (OIDC BCL 1.0)
	f.StringVar(&rootArgs.config.BackchannelLogoutURI, "backchannel-logout-uri", "", "URL to POST a signed logout_token to when users log out successfully. Leave empty (default) to disable back-channel logout notifications. See OIDC Back-Channel Logout 1.0.")

	// OpenFGA fine-grained authorization. By default FGA reuses the main database
	// when it is sqlite/postgres/mysql/mariadb (no extra config needed). These
	// flags override that — required only when the main DB is unsupported
	// (mongodb, dynamodb, …) or to use a dedicated FGA store.
	f.StringVar(&rootArgs.config.FGAStore, "fga-store", "", "Override the OpenFGA datastore: 'sqlite', 'postgres', 'mysql', or 'memory' (dev). Default: reuse the main database when it is SQL-compatible; required only for unsupported main DBs (mongodb, dynamodb, …)")
	f.StringVar(&rootArgs.config.FGAStoreURL, "fga-store-url", "", "Connection URI for an overridden --fga-store (file: URI for sqlite, DSN for postgres/mysql). Ignored when FGA reuses the main database")

	// Deprecated flags
	_ = f.MarkDeprecated("database_url", "use --database-url instead")
	_ = f.MarkDeprecated("database_type", "use --database-type instead")
	_ = f.MarkDeprecated("env_file", "no more supported")
	_ = f.MarkDeprecated("log_level", "use --log-level instead")
	_ = f.MarkDeprecated("redis_url", "use --redis-url instead")
}

// applyFlagDefaults sets config and server fields to their flag defaults when the
// value is empty (e.g. when user passes --host="" we use the default from vars above).
func applyFlagDefaults() {
	c := &rootArgs.config
	s := &rootArgs.server

	if s.HTTPPort == 0 {
		s.HTTPPort = defaultHTTPPort
	}
	if s.MetricsPort == 0 {
		s.MetricsPort = defaultMetricsPort
	}
	if strings.TrimSpace(s.MetricsHost) == "" {
		s.MetricsHost = defaultMetricsHost
	}
	if strings.TrimSpace(rootArgs.logLevel) == "" {
		rootArgs.logLevel = defaultLogLevel
	}
	if strings.TrimSpace(c.OrganizationLogo) == "" {
		c.OrganizationLogo = defaultOrganizationLogo
	}
	if strings.TrimSpace(c.OrganizationName) == "" {
		c.OrganizationName = defaultOrganizationName
	}
	if strings.TrimSpace(c.DefaultAuthorizeResponseMode) == "" {
		c.DefaultAuthorizeResponseMode = constants.ResponseModeQuery
	}
	if strings.TrimSpace(c.DefaultAuthorizeResponseType) == "" {
		c.DefaultAuthorizeResponseType = constants.ResponseTypeToken
	}
	// AdminSecret deliberately has NO default. The fatal check in runRoot
	// rejects empty values at startup; secret strength is the operator's
	// responsibility.
	if len(c.AllowedOrigins) == 0 {
		c.AllowedOrigins = append([]string(nil), defaultAllowedOrigins...)
	}
	if len(c.DefaultRoles) == 0 {
		c.DefaultRoles = append([]string(nil), defaultRoles...)
	}
	if len(c.Roles) == 0 {
		c.Roles = append([]string(nil), defaultRoles...)
	}
	if strings.TrimSpace(c.JWTRoleClaim) == "" {
		c.JWTRoleClaim = defaultJWTRoleClaim
	}
	if strings.TrimSpace(c.MicrosoftTenantID) == "" {
		c.MicrosoftTenantID = defaultMicrosoftTenantID
	}
	if len(c.GoogleScopes) == 0 {
		c.GoogleScopes = append([]string(nil), defaultGoogleScopes...)
	}
	if len(c.GithubScopes) == 0 {
		c.GithubScopes = append([]string(nil), defaultGithubScopes...)
	}
	if len(c.FacebookScopes) == 0 {
		c.FacebookScopes = append([]string(nil), defaultFacebookScopes...)
	}
	if len(c.MicrosoftScopes) == 0 {
		c.MicrosoftScopes = append([]string(nil), defaultMicrosoftScopes...)
	}
	if len(c.TwitchScopes) == 0 {
		c.TwitchScopes = append([]string(nil), defaultTwitchScopes...)
	}
	if len(c.LinkedinScopes) == 0 {
		c.LinkedinScopes = append([]string(nil), defaultLinkedinScopes...)
	}
	if len(c.AppleScopes) == 0 {
		c.AppleScopes = append([]string(nil), defaultAppleScopes...)
	}
	if len(c.DiscordScopes) == 0 {
		c.DiscordScopes = append([]string(nil), defaultDiscordScopes...)
	}
	if len(c.TwitterScopes) == 0 {
		c.TwitterScopes = append([]string(nil), defaultTwitterScopes...)
	}
	if len(c.RobloxScopes) == 0 {
		c.RobloxScopes = append([]string(nil), defaultRobloxScopes...)
	}
}

// Run the service
func runRoot(c *cobra.Command, args []string) {
	applyFlagDefaults()
	// All three listeners (HTTP, metrics, gRPC) bind concurrently; any
	// collision is unrecoverable at runtime, so we fail fast at startup.
	ports := map[string]int{
		"--http-port":    rootArgs.server.HTTPPort,
		"--metrics-port": rootArgs.server.MetricsPort,
		"--grpc-port":    rootArgs.config.GRPCPort,
	}
	for nameA, a := range ports {
		for nameB, b := range ports {
			if nameA < nameB && a == b {
				fmt.Fprintf(os.Stderr, "invalid server ports: %s (%d) and %s (%d) must differ — each listener binds independently\n", nameA, a, nameB, b)
				os.Exit(1)
			}
		}
	}

	if err := validateAuthorizerURL(&rootArgs.config); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if err := validateMCPConfig(&rootArgs.config); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if err := validateRedirectURIs(&rootArgs.config); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	// Refuse to start without an admin secret. The previous default of
	// "password" was a publicly known credential — operators upgrading from
	// older versions must now supply --admin-secret explicitly. The strength
	// of the supplied value is the operator's responsibility; we only
	// guarantee it is non-empty.
	if strings.TrimSpace(rootArgs.config.AdminSecret) == "" {
		fmt.Fprintln(os.Stderr, "FATAL: --admin-secret is required and must not be empty.")
		os.Exit(1)
	}

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

	// Warn if AllowedOrigins is the wildcard ["*"] — this is a development-
	// friendly default but in production it pairs poorly with credentialed
	// requests. Operators should set an explicit allowlist before deploying.
	for _, o := range rootArgs.config.AllowedOrigins {
		if o == "*" {
			log.Warn().Msg("AllowedOrigins contains \"*\" — this is unsafe for production. Set --allowed-origins to an explicit list of trusted origins. CSRF middleware will fall back to same-origin enforcement for state-changing requests.")
			break
		}
	}

	// Warn if --env=e2e is set — this relaxes SSRF protection (private/loopback-IP
	// rejection) for the per-org SSO broker and webhook registration/delivery, and
	// routes social-login OAuth and SMS sending to fixed e2e-playground mock
	// addresses. It exists solely for e2e-playground/docker-compose.yml and must
	// never be set on a real deployment; unlike production/staging, there is no
	// other visible symptom if it is (see internal/constants/env.go's E2EEnv doc).
	if rootArgs.config.Env == constants.E2EEnv {
		log.Warn().Msg("running with --env=e2e: SSRF protection is relaxed for the SSO broker and webhooks, and OAuth/SMS are routed to e2e-playground mock addresses. This must never be set in a real deployment.")
	}

	// Warn when the at-rest key is only the JWT secret by fallback. This is
	// cryptographically fine — JWTSecret is a real secret — but it couples two
	// keys with opposite lifecycles. A signing key is meant to be rotatable and
	// rotation is cheap (tokens expire, users log in again); the at-rest key has
	// NO re-encryption path, so while they are the same value, rotating
	// --jwt-secret silently makes every enrolled TOTP authenticator
	// undecryptable and invalidates outstanding OTPs.
	//
	// A warning rather than a hard failure, deliberately: unlike an EMPTY key
	// (which is rejected outright in Config.ValidateEncryptionKey because the
	// data is unprotected *now*), the data here IS protected. The risk is a
	// future operator action, and this is the only chance to inform that
	// decision before enrolments exist and make the fix expensive.
	if rootArgs.config.EncryptionKey == rootArgs.config.JWTSecret {
		log.Warn().Msg("--encryption-key is not set and has fallen back to --jwt-secret. Secrets at rest (TOTP seeds, OTP digests) are keyed by the same value that signs tokens, so rotating --jwt-secret will lock out every enrolled TOTP user — there is no re-encryption path. Set a distinct --encryption-key now; doing it after users enrol requires them to re-enrol.")
	}

	// Email verification with no way to send email is an unrecoverable trap, not
	// a degraded mode: signup creates the account unverified, the verification
	// mail never leaves, and every self-service route out of that state (the
	// signup link, resend-verification, the login email-OTP fallback) is the
	// same mailbox. The user is stranded permanently, and an unverified account
	// also blocks a federated login for the same address. Fail at boot, where
	// the operator can see it, rather than silently per-user.
	if rootArgs.config.EnableEmailVerification && !rootArgs.config.IsEmailServiceEnabled {
		log.Fatal().Msg("--enable-email-verification=true requires a working email service, but SMTP is not configured. Users would be created unverified with no way to ever verify. Set --smtp-host, --smtp-port and --smtp-sender-email, or disable email verification.")
	}

	// The compatibility escape hatch for unattested federated emails. It is
	// narrowed (an unattested address still cannot cross into an account another
	// credential owns), but it leaves same-provider collisions open — two Entra
	// tenants asserting one address. Warn every boot so it does not quietly
	// become permanent.
	if rootArgs.config.OAuthAllowUnverifiedProviderEmail {
		log.Warn().Msg("--oauth-allow-unverified-provider-email is set: a social login whose provider does not attest the email address may still sign up or return to an account that same provider owns. Cross-credential takeover is still blocked, but two principals of the SAME provider (e.g. two Entra tenants) can collide on one address. Pin --microsoft-tenant-id, set --microsoft-allowed-tenants, or enable the xms_edov optional claim, then remove this flag. See docs/email-verification-contract.md.")
	}

	// Initialize prometheus metrics
	metrics.Init()

	// Service availability and MFA defaults are derived in Config.Finalize(),
	// invoked from the root command's PersistentPreRun.

	// Canonicalize Config.ClientID once at load so the seeded registry client_id
	// (which is trimmed) is byte-identical to every remaining raw Config.ClientID
	// comparison — introspection audience, revocation ownership, client_check, and
	// the /app bootstrap all key on the exact same string. Closes the whitespace
	// edge where a padded --client-id would seed a trimmed row but compare raw.
	rootArgs.config.ClientID = strings.TrimSpace(rootArgs.config.ClientID)

	// Pin the trusted host used for all self-referential URLs (email links,
	// JWT iss, OIDC discovery) BEFORE any listener starts. When --url is set,
	// request headers can no longer control the server's own base URL, closing
	// the host-header-injection account-takeover class (CWE-640). Empty keeps
	// the legacy header-based derivation.
	parsers.SetTrustedURL(rootArgs.config.AuthorizerURL)
	parsers.SetLogger(&log)
	if strings.TrimSpace(rootArgs.config.AuthorizerURL) == "" {
		// GHSA-m82j-rq33-qjx2 (CWE-640, CVSS 8.1). With --url unset, the base URL
		// for emailed reset/verification/magic links and the JWT `iss` is derived
		// from request headers (X-Authorizer-URL, X-Forwarded-Host, Host). An
		// unauthenticated attacker can therefore poison the reset link sent to a
		// victim, and — because redemption re-validates `iss` against the SAME
		// request-derived host — replay the identical spoofed header to redeem the
		// stolen token. The attacker controls both sides of that comparison, so it
		// is no boundary at all.
		//
		// Warned, not fatal: refusing to boot would break every existing
		// deployment on upgrade. Making --url mandatory is the real fix and is a
		// deliberate follow-up, so this must be loud enough that nobody reaches
		// production without seeing it.
		log.Warn().
			Str("advisory", "GHSA-m82j-rq33-qjx2").
			Str("cwe", "CWE-640").
			Str("fix", "start Authorizer with --url=https://your-authorizer-host").
			Msg("SECURITY: --url is not set. Password-reset, email-verification and magic-link URLs will be built from attacker-controllable request headers (Host / X-Forwarded-Host / X-Authorizer-URL), allowing reset-link poisoning and account takeover. Set --url to your canonical base URL in any deployment that sends email.")
	}

	// Storage provider
	storageProvider, err := storage.New(&rootArgs.config, &storage.Dependencies{
		Log: &log,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create storage provider")
	}
	defer func() {
		if err := storageProvider.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close storage provider")
		}
	}()

	// Seed the reserved interactive client (Config.ClientID) into the client
	// registry. Idempotent and non-fatal — must run after storage is up and
	// before the Token/HTTP subsystems that will (in a later PR) resolve client
	// auth from this row.
	seedReservedClient(context.Background(), storageProvider, &rootArgs.config, &log)

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
		Log:             &log,
		StorageProvider: storageProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create memory store provider")
	}

	// Authenticator provider — depends on the memory store for the transient
	// pending-secret used by safe TOTP re-enrollment (see totp.Generate).
	authenticatorProvider, err := authenticators.New(&rootArgs.config, &authenticators.Dependencies{
		Log:                 &log,
		StorageProvider:     storageProvider,
		MemoryStoreProvider: memoryStoreProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create authenticator provider")
	}

	// WebAuthn/passkey provider
	webAuthnProvider, err := webauthn.NewProvider(&webauthn.Dependencies{
		Log:                 &log,
		StorageProvider:     storageProvider,
		MemoryStoreProvider: memoryStoreProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create webauthn provider")
	}

	// Rate limit provider
	rateLimitDeps := &rate_limit.Dependencies{
		Log: &log,
	}
	// If memory store is Redis-backed, reuse its client for distributed rate limiting
	type redisClientProvider interface {
		Client() interface {
			Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
		}
	}
	if rcp, ok := memoryStoreProvider.(redisClientProvider); ok {
		if client, ok := rcp.Client().(rate_limit.RedisClient); ok {
			rateLimitDeps.RedisStore = client
		}
	}
	rateLimitProvider, err := rate_limit.New(&rootArgs.config, rateLimitDeps)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create rate limit provider")
	}
	defer func() { _ = rateLimitProvider.Close() }()

	// Embedded OpenFGA authorization engine (optional; nil when --fga-store
	// is not configured). NOTE: multi-replica deployments should prefer
	// running migrations once via an init job — concurrent on-boot migrations
	// rely on the migration tool's own locking and add cold-start latency.
	authzEngine, closeAuthzEngine := initAuthzEngine(&rootArgs.config, &log)
	defer closeAuthzEngine()

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
		StorageProvider:     storageProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create token provider")
	}
	// OAuth provider
	oauthProvider, err := oauth.New(&rootArgs.config, &oauth.Dependencies{
		Log: &log,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create oauth provider")
	}

	// Ensure client ID and secret are set for authorizer instance
	if strings.TrimSpace(rootArgs.config.ClientID) == "" {
		log.Fatal().Msg("client ID missing in rootArgs")
	}

	if strings.TrimSpace(rootArgs.config.ClientSecret) == "" {
		log.Fatal().Msg("client secret missing in rootArgs")
	}

	auditProvider := audit.New(&audit.Dependencies{
		Log:             &log,
		StorageProvider: storageProvider,
	})

	// Transport-agnostic service layer that hosts public-API operations.
	// GraphQL, gRPC, and REST surfaces all delegate to this.
	serviceProvider, err := service.New(&rootArgs.config, &service.Dependencies{
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
		RateLimitProvider:     rateLimitProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create service provider")
	}

	// CIMD resolver. nil when disabled, which is what switches the feature off
	// everywhere downstream — the authorize handler and the client-auth resolver
	// both treat a nil provider as "URL client_ids are not a thing here".
	var clientMetadataProvider *clientmetadata.Provider
	if rootArgs.config.EnableClientIDMetadataDocument {
		clientMetadataProvider = clientmetadata.New(&log, rootArgs.config.ClientIDMetadataAllowedDomains,
			rootArgs.config.Env == constants.E2EEnv)
		log.Info().Strs("allowed_domains", rootArgs.config.ClientIDMetadataAllowedDomains).
			Msg("Client ID Metadata Documents enabled")
	}

	httpProvider, err := http_handlers.New(&rootArgs.config, &http_handlers.Dependencies{
		ClientMetadataProvider: clientMetadataProvider,
		Log:                    &log,
		AuditProvider:          auditProvider,
		AuthenticatorProvider:  authenticatorProvider,
		EmailProvider:          emailProvider,
		EventsProvider:         eventsProvider,
		MemoryStoreProvider:    memoryStoreProvider,
		SMSProvider:            smsProvider,
		StorageProvider:        storageProvider,
		TokenProvider:          tokenProvider,
		OAuthProvider:          oauthProvider,
		RateLimitProvider:      rateLimitProvider,
		ServiceProvider:        serviceProvider,
		AuthzEngine:            authzEngine,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create http provider")
	}

	// gRPC server — listens on --grpc-port. The REST gateway built by
	// server.Run wraps this same gRPC server in-process so /v1/* REST
	// calls translate to local gRPC method invocations (no network hop).
	grpcAddr := net.JoinHostPort(rootArgs.server.Host, strconv.Itoa(rootArgs.config.GRPCPort))
	grpcSrv, err := grpcsrv.New(grpcAddr, &grpcsrv.Dependencies{
		Log:             &log,
		Config:          &rootArgs.config,
		ServiceProvider: serviceProvider,
		TokenProvider:   tokenProvider,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create grpc server")
	}
	rootArgs.server.GRPCPort = rootArgs.config.GRPCPort

	// MCP surface, served at POST /mcp on the main HTTP listener when enabled.
	//
	// It gets its OWN gRPC server rather than reusing grpcSrv above, and that is
	// the security boundary, not an implementation detail. This one never binds a
	// port — the MCP handler dials it over an in-process bufconn — and its auth
	// interceptor accepts exactly one kind of credential: a bearer token whose
	// audience is this deployment's canonical <url>/mcp. The port-listening server
	// rejects that audience, and this one rejects everything the port-listening
	// server accepts. Two objects, so a token minted for one surface cannot
	// authenticate the other by construction rather than by a conditional.
	//
	// Every provider is shared with the main server. The `authorizer mcp`
	// subcommand builds a second copy of the entire stack — storage, memory
	// store, FGA engine and all — which is precisely what made it undeployable.
	var mcpHandler http.Handler
	if rootArgs.config.MCPEnabled {
		mcpResource := rootArgs.config.MCPResource()
		mcpGRPC, mErr := grpcsrv.New(":0", &grpcsrv.Dependencies{
			Log:             &log,
			Config:          &rootArgs.config,
			ServiceProvider: serviceProvider,
			TokenProvider:   tokenProvider,
			TokenResolver:   interceptors.MCPTokenResolver(tokenProvider, mcpResource),
		})
		if mErr != nil {
			log.Fatal().Err(mErr).Msg("failed to create mcp grpc server")
		}
		mcpSrv, mErr := mcp.New(&log, mcpGRPC.GRPCServer(), mcp.Options{
			Name:    "authorizer",
			Version: constants.VERSION,
		})
		if mErr != nil {
			log.Fatal().Err(mErr).Msg("failed to create mcp server")
		}
		mcpHandler = mcpSrv.Handler()
		log.Info().Str("resource", mcpResource).Msg("MCP enabled at POST /mcp")
	}

	// Inbound SCIM 2.0 server (per-org user provisioning). Transport-thin
	// handler over the scim service; org resolved only from the bearer token.
	scimService := scim.New(&scim.Dependencies{
		Log:                 &log,
		StorageProvider:     storageProvider,
		MemoryStoreProvider: memoryStoreProvider,
		AuthzEngine:         authzEngine,
		EventsProvider:      eventsProvider,
	})
	scimHandler := scimhttp.New(&scimhttp.Dependencies{
		Log:     &log,
		Service: scimService,
	})

	// Prepare server
	deps := &server.Dependencies{
		Log:          &log,
		AppConfig:    &rootArgs.config,
		HTTPProvider: httpProvider,
		ScimHandler:  scimHandler,
		MCPHandler:   mcpHandler,
		GRPCServer:   grpcSrv,
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

// validateAuthorizerURL refuses to start without a usable canonical base URL.
//
// Without --url, every self-referential URL this server emits — the password
// reset link, the email verification link, the magic link, the JWT `iss` claim,
// the OIDC discovery and JWKS URLs — is derived from REQUEST HEADERS
// (X-Authorizer-URL, then X-Forwarded-Host, then Host). An unauthenticated
// attacker can therefore send a forgot-password request carrying their own
// Host, and the victim receives a genuine reset link pointing at the attacker's
// domain. When the victim clicks it the token is handed over, and because the
// token's `iss` is validated against that same header-derived host, the
// attacker redeems it by replaying the same spoofed Host. Full account takeover,
// no prior access, no mailbox compromise (CWE-640).
//
// Making --url mandatory is the only fix that closes the class. Validating the
// derived host against the origin allowlist would help only deployments that
// configured an explicit list, and would do nothing on the default "*" — which
// is the configuration the attack targets.
//
// This costs no supported capability. Setting --url ALREADY collapses an
// instance to a single canonical host (GetHostFromRequest returns it and ignores
// every header), so multi-host operation only ever worked on the vulnerable
// path. Verified org domains are email-domain-to-organization routing for home
// realm discovery, not HTTP virtual hosting, and are unaffected.
//
// An unusable value is rejected, not just an empty one: SetTrustedURL treats
// anything sanitizeAuthorizerURL cannot normalize as UNSET and falls back to
// headers, so "--url=auth.example.com" or "--url=https://user:pw@host" would
// otherwise start in the vulnerable configuration while looking configured.
func validateAuthorizerURL(cfg *config.Config) error {
	if strings.TrimSpace(cfg.AuthorizerURL) == "" {
		return fmt.Errorf("--url is required (e.g. --url=https://auth.example.com)\n\n" +
			"  Why: without it the password-reset, email-verification and magic-link URLs, and the\n" +
			"  JWT `iss` claim, are derived from request headers — so an attacker can have a victim\n" +
			"  emailed a genuine reset link pointing at a domain the attacker controls.\n\n" +
			"  Note --url is NOT --allowed-origins; you need both:\n" +
			"    --url             this server's own address, e.g. https://auth.example.com\n" +
			"    --allowed-origins your apps this server may redirect to, e.g. https://app.example.com")
	}
	if parsers.SanitizeAuthorizerURL(cfg.AuthorizerURL) == "" {
		return fmt.Errorf("--url=%q is not a usable canonical URL: it must be an absolute http(s) URL "+
			"with a host and no user info (e.g. https://auth.example.com). An unusable value is "+
			"treated as unset, which silently restores header-derived URLs", cfg.AuthorizerURL)
	}
	return nil
}

// validateRedirectURIs refuses --redirect-uris entries that could never match.
//
// The comparison this list feeds is exact (see redirectURIMatches), so a value
// that is not a usable absolute http(s) URL cannot match anything a browser
// would present. Left unchecked it does not fail loudly: the flag looks
// configured, every login is refused with "invalid redirect_uri", and nothing
// says which entry is at fault.
//
// A fragment or user info is rejected outright for the same reason
// redirectURIMatches rejects them — a fragment swallows the authorization
// response, user info is the "evil.com@real.host" phishing shape — so a
// registration carrying either is a mistake, not a match that never fires.
func validateRedirectURIs(cfg *config.Config) error {
	for _, raw := range cfg.RedirectURIs {
		uri := strings.TrimSpace(raw)
		if uri == "" {
			continue
		}
		u, err := url.Parse(uri)
		switch {
		case err != nil:
			return fmt.Errorf("--redirect-uris entry %q is not a valid URL: %w", uri, err)
		case u.Scheme != "http" && u.Scheme != "https":
			return fmt.Errorf("--redirect-uris entry %q must be an absolute http(s) URL "+
				"(e.g. https://app.example.com/callback)", uri)
		case u.Host == "":
			return fmt.Errorf("--redirect-uris entry %q has no host", uri)
		case u.Fragment != "":
			return fmt.Errorf("--redirect-uris entry %q carries a fragment; RFC 6749 §3.1.2 "+
				"forbids one on the redirection endpoint", uri)
		case u.User != nil:
			return fmt.Errorf("--redirect-uris entry %q carries user info, which reads as one "+
				"host while resolving to another", uri)
		}
	}
	return nil
}

// validateMCPConfig refuses a configuration that would enable MCP without a
// usable canonical URL.
//
// This is the second lock on the audience door, and it is not optional. MCP's
// entire security model is one comparison: a token is accepted at /mcp only if
// its `aud` equals this deployment's canonical <url>/mcp. Without --url that
// identifier would be derived from request headers — parsers.GetHost falls back
// to X-Authorizer-URL, then X-Forwarded-Host, then Host — so the caller would be
// supplying both sides of the comparison and there would be no check at all.
//
// It also rejects a --url that is merely unusable (no scheme, userinfo, a
// non-http scheme), because MCPResource() returns empty for those too. Starting
// anyway would produce a surface that is enabled, advertises nothing, and
// rejects every token: broken in a way that reports success.
//
// Extracted from runRoot so it can be tested. The inline version behind
// os.Exit(1) could be rewritten into something weaker — comparing AuthorizerURL
// to "" instead of asking whether a resource can be derived from it — with the
// whole suite staying green.
func validateMCPConfig(cfg *config.Config) error {
	if !cfg.MCPEnabled {
		return nil
	}
	if cfg.MCPResource() == "" {
		return fmt.Errorf("--mcp-enabled requires a valid --url (e.g. https://auth.example.com): " +
			"the MCP resource identifier that access tokens are bound to is derived from it, and " +
			"deriving it from request headers instead would let a caller choose their own audience")
	}
	return nil
}

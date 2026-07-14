package config

import "strings"

// Config defines the configuration for the authorizer instance
type Config struct {
	// Env is the environment of the authorizer instance
	Env string
	// SkipTestEndpointSSRFValidation relaxes SSRF checks for the admin TestEndpoint GraphQL
	// mutation (e.g. to hit localhost in tests). Must remain false in production; integration
	// tests enable it together with Env=test.
	SkipTestEndpointSSRFValidation bool
	// OrganizationLogo is the logo of the organization
	OrganizationLogo string
	// OrganizationName is the name of the organization
	OrganizationName string
	// AdminSecret is the secret for the admin
	AdminSecret string
	// AllowedOrigins is the list of allowed origins
	AllowedOrigins []string

	// EnableLoginPage is the flag to enable login page
	EnableLoginPage bool
	// EnableOrgDiscovery gates the public home-realm-discovery endpoint
	// (GET /api/v1/org-discovery) AND the /app email-first SSO routing step.
	// OPT-IN, off by default: most deployments are single-tenant with no
	// enterprise SSO, so the login page must stay unchanged until a
	// multi-tenant operator turns this on. Surfaced on Meta so the SPA knows.
	EnableOrgDiscovery bool
	// EnablePlayground is the flag to enable playground
	EnablePlayground bool
	// EnableGraphQLIntrospection is the flag to enable GraphQL introspection
	EnableGraphQLIntrospection bool
	// EnableHSTS opts in to the Strict-Transport-Security response header.
	// Off by default because operators not behind TLS would lock themselves
	// out for a year. Enable in production behind TLS.
	EnableHSTS bool
	// DisableCSP turns off the default Content-Security-Policy header.
	// Off by default — CSP is on by default. Provided as an escape hatch
	// for dashboards that load assets in ways the default policy blocks.
	DisableCSP bool
	// GraphQLMaxComplexity caps the total complexity score of a single GraphQL
	// operation. Operations exceeding this limit are rejected before execution.
	GraphQLMaxComplexity int
	// GraphQLMaxDepth caps the maximum nesting depth of a GraphQL selection set.
	GraphQLMaxDepth int
	// GraphQLMaxAliases caps the total number of aliased fields per operation.
	// Defends against alias-amplification denial-of-service attacks.
	GraphQLMaxAliases int
	// GraphQLMaxBodyBytes caps the size of the request body accepted by the
	// GraphQL endpoint to prevent oversized-payload denial of service.
	GraphQLMaxBodyBytes int64

	// gRPC server configuration
	// GRPCPort is the port the gRPC server listens on.
	GRPCPort int
	// EnableGRPCReflection toggles the gRPC server-reflection service.
	// Default: on (matches the playground). Disable in locked-down prod.
	EnableGRPCReflection bool
	// GRPCTLSCert / GRPCTLSKey set the TLS material for the gRPC listener.
	// When unset and GRPCInsecure is false, the server refuses to start.
	GRPCTLSCert string
	GRPCTLSKey  string
	// GRPCInsecure permits cleartext gRPC. For local dev only; production
	// should always set TLS material.
	GRPCInsecure bool

	// Database Configurations
	// DatabaseType is the type of database to use
	DatabaseType string
	// DatabaseURL is the URL of the database
	DatabaseURL string
	// DatabaseName is the name of the database
	DatabaseName string
	// DatabaseUsername is the username for the database
	DatabaseUsername string
	// DatabasePassword is the password for the database
	DatabasePassword string
	// DatabaseHost is the host for the database
	DatabaseHost string
	// DatabasePort is the port for the database
	DatabasePort int
	// DatabaseCert is the certificate for the database
	DatabaseCert string
	// DatabaseCACert is the CA certificate for the database
	DatabaseCACert string
	// DatabaseCertKey is the certificate key for the database
	DatabaseCertKey string

	// CouchBase flags
	// CouchBaseBucket is the bucket for the database
	// Used only for CouchBase database
	CouchBaseBucket string
	// CouchBaseRamQuota is the RAM quota for the database
	// Used only for CouchBase database
	CouchBaseRamQuota string
	// CouchBaseScope is the scope for the database
	// Used only for CouchBase database
	CouchBaseScope string
	// CouchBaseWaitTimeout is the timeout in seconds for Couchbase WaitUntilReady operations
	// Used only for CouchBase database
	CouchBaseWaitTimeout int

	// AWS flags
	// AWSRegion is the region for the database
	// Used only for AWS database
	AWSRegion string
	// AWSAccessKeyID is the access key ID for the database
	// Used only for AWS database
	AWSAccessKeyID string
	// AWSSecretAccessKey is the secret access key for the database
	// Used only for AWS database
	AWSSecretAccessKey string

	// Email Configurations
	// SMTPHost is the host for the SMTP server
	SMTPHost string
	// SMTPPort is the port for the SMTP server
	SMTPPort int
	// SMTPUsername is the username for the SMTP server
	SMTPUsername string
	// SMTPPassword is the password for the SMTP server
	SMTPPassword string
	// SMTPSenderEmail is the sender email for the SMTP server
	SMTPSenderEmail string
	// SMTPSenderName is the sender name for the SMTP server
	SMTPSenderName string
	// SMTPLocalName is the local name for the SMTP server
	SMTPLocalName string
	// SkipTLSVerification is the flag to skip TLS verification
	SMTPSkipTLSVerification bool

	// Memory Store Configurations
	// RedisURL is the URL of the redis server
	RedisURL string

	// Auth Configurations
	// DefaultRoles is the default roles for the user
	// It is a comma separated string
	// TODO: check derived keys
	DefaultRoles []string
	// Roles is the list of all the roles of the user
	// It is a comma separated string
	Roles []string
	// ProtectedRoles is the list of all the protected roles
	// For this roles, sign-up is disabled
	// It is a comma separated string
	ProtectedRoles []string
	// EnableStrongPassword is the flag to enable strong password
	EnableStrongPassword bool
	// EnableTOTPLogin boolean to enable TOTP login. Derived from DisableTOTPLogin.
	EnableTOTPLogin bool
	// EnableBasicAuthentication boolean to enable basic authentication
	EnableBasicAuthentication bool
	// EnableMagicLinkLogin boolean to enable magic link login
	EnableMagicLinkLogin bool
	// EnableEmailVerification boolean to enable email verification
	EnableEmailVerification bool
	// EnableMobileBasicAuthentication boolean to enable mobile basic authentication
	EnableMobileBasicAuthentication bool
	// EnablePhoneVerification boolean to enable phone verification
	EnablePhoneVerification bool
	// EnableMFA is derived (see Finalize): true when at least one MFA method is
	// usable. Not set from a flag.
	EnableMFA bool
	// EnableEmailOTP boolean to enable email OTP. Derived from DisableEmailOTP.
	EnableEmailOTP bool
	// EnableSMSOTP boolean to enable SMS OTP. Derived from DisableSMSOTP.
	EnableSMSOTP bool
	// EnableWebauthnMFA is derived from DisableWebauthnMFA — whether a
	// registered WebAuthn/passkey credential counts as an MFA factor. This is
	// the ONLY WebAuthn-MFA flag; "webauthn" and "passkey" are the same
	// credential type in this codebase (see spec), not two separate factors.
	EnableWebauthnMFA bool
	// DisableTOTPLogin opts out of TOTP MFA (enabled by default).
	DisableTOTPLogin bool
	// DisableEmailOTP opts out of email OTP MFA (enabled by default when the
	// email service is configured).
	DisableEmailOTP bool
	// DisableSMSOTP opts out of SMS OTP MFA (enabled by default when the SMS
	// service is configured).
	DisableSMSOTP bool
	// DisableWebauthnMFA opts out of WebAuthn/passkey as an MFA factor
	// (enabled by default). Does not affect WebAuthn/passkey as a PRIMARY
	// login method (the passkey button on the login screen) — that is a
	// separate, pre-existing feature, untouched by this flag.
	DisableWebauthnMFA bool
	// DisableMFA is a one-way global kill switch: when set, Finalize forces
	// EnableMFA and EnforceMFA off regardless of the per-method flags. It can
	// only ever turn MFA off, so unlike the removed --enable-mfa it cannot
	// contradict the per-method flags. Does not affect WebAuthn/passkey as a
	// PRIMARY login method — only its role as an MFA factor.
	DisableMFA bool
	// EnableSignup boolean to enable signup
	EnableSignup bool
	// IsEmailServiceEnabled is derived from SMTP configurations
	IsEmailServiceEnabled bool
	// IsSMSServiceEnabled is derived from Twilio configurations
	IsSMSServiceEnabled bool
	// EnforceMFA is the flag to enforce MFA
	EnforceMFA bool

	// URLs
	// ResetPasswordURL is the URL for reset password
	ResetPasswordURL string

	// JWT Configurations
	// JWTType is the type of JWT to use
	JWTType string
	// JWTSecret is the secret for the JWT
	JWTSecret string
	// JWTPublicKey is the public key for the JWT
	JWTPublicKey string
	// JWTPrivateKey is the private key for the JWT
	JWTPrivateKey string
	// JWTSecondaryType is the algorithm of an optional secondary JWT
	// key used for manual key rotation. When set along with the other
	// JWT secondary fields, the JWKS endpoint will publish both keys
	// and token validation will accept tokens signed with either key.
	// New tokens are always signed with the primary key (JWTType).
	// Leave empty to disable multi-key mode (default).
	JWTSecondaryType string
	// JWTSecondarySecret is the secret for the secondary JWT key.
	// Used only when JWTSecondaryType is an HMAC algorithm. HMAC keys
	// are never exposed via the JWKS endpoint.
	JWTSecondarySecret string
	// JWTSecondaryPublicKey is the public key for the secondary JWT
	// key. Used when JWTSecondaryType is RSA or ECDSA.
	JWTSecondaryPublicKey string
	// JWTSecondaryPrivateKey is the private key for the secondary JWT
	// key. Currently unused at the signing stage (the primary key is
	// always used to sign); kept for symmetry and for future
	// primary/secondary swap automation.
	JWTSecondaryPrivateKey string
	// JWTRoleClaim is the role claim for the JWT
	JWTRoleClaim string
	// RefreshTokenExpiresIn is the refresh token lifetime in seconds.
	// Defaults to 30 days (2592000 seconds) when zero or unset.
	RefreshTokenExpiresIn int64
	// CustomAccessTokenScript is the custom access token script
	CustomAccessTokenScript string

	// OAuth Configurations
	// ClientID is the client ID for the authorizer
	ClientID string
	// ClientSecret is the secret for the authorizer
	ClientSecret string
	// Default Authorize response mode
	DefaultAuthorizeResponseMode string
	// Default Authorize response type
	DefaultAuthorizeResponseType string

	// Twilio Configurations
	// TwilioAPISecret is the API secret for Twilio
	TwilioAPISecret string
	// TwilioAPIKey is the API key for Twilio
	TwilioAPIKey string
	// TwilioSender is the sender for Twilio
	TwilioSender string
	// TwilioAccountSID is the account SID for Twilio
	TwilioAccountSID string

	// OAuth providers that authorizer supports
	// GoogleClientID is the client ID for Google OAuth
	GoogleClientID string
	// GoogleClientSecret is the client secret for Google OAuth
	GoogleClientSecret string
	// Scopes is the list of scopes for Google OAuth
	GoogleScopes []string

	// GithubClientID is the client ID for Github OAuth
	GithubClientID string
	// GithubClientSecret is the client secret for Github OAuth
	GithubClientSecret string
	// GithubScopes is the list of scopes for Github OAuth
	GithubScopes []string

	// FacebookClientID is the client ID for Facebook OAuth
	FacebookClientID string
	// FacebookClientSecret is the client secret for Facebook OAuth
	FacebookClientSecret string
	// FacebookScopes is the list of scopes for Facebook OAuth
	FacebookScopes []string

	// LinkedinClientID is the client ID for Linkedin OAuth
	LinkedinClientID string
	// LinkedinClientSecret is the client secret for Linkedin OAuth
	LinkedinClientSecret string
	// LinkedinScopes is the list of scopes for Linkedin OAuth
	LinkedinScopes []string

	// TwitterClientID is the client ID for Twitter OAuth
	TwitterClientID string
	// TwitterClientSecret is the client secret for Twitter OAuth
	TwitterClientSecret string
	// TwitterScopes is the list of scopes for Twitter OAuth
	TwitterScopes []string

	// MicrosoftClientID is the client ID for Microsoft OAuth
	MicrosoftClientID string
	// MicrosoftClientSecret is the client secret for Microsoft OAuth
	MicrosoftClientSecret string
	// MicrosoftTenantID is the tenant ID for Microsoft OAuth
	MicrosoftTenantID string
	// MicrosoftScopes is the list of scopes for Microsoft OAuth
	MicrosoftScopes []string

	// AppleClientID is the client ID for Apple OAuth
	AppleClientID string
	// AppleClientSecret is the client secret for Apple OAuth
	AppleClientSecret string
	// AppleScopes is the list of scopes for Apple OAuth
	AppleScopes []string

	// DiscordClientID is the client ID for Discord OAuth
	DiscordClientID string
	// DiscordClientSecret is the client secret for Discord OAuth
	DiscordClientSecret string
	// DiscordScopes is the list of scopes for Discord OAuth
	DiscordScopes []string

	// TwitchClientID is the client ID for Twitch OAuth
	TwitchClientID string
	// TwitchClientSecret is the client secret for Twitch OAuth
	TwitchClientSecret string
	// TwitchScopes is the list of scopes for Twitch OAuth
	TwitchScopes []string

	// RobloxClientID is the client ID for Roblox OAuth
	RobloxClientID string
	// RobloxClientSecret is the client secret for Roblox OAuth
	RobloxClientSecret string
	// RobloxScopes is the list of scopes for Roblox OAuth
	RobloxScopes []string

	// IsAppCookieSecure is the flag to set secure(http only) cookie
	AppCookieSecure bool
	// AppCookieSameSite controls the SameSite attribute for session cookies.
	// Valid values: "none" (default), "lax", "strict".
	// "none" preserves backward compatibility for cross-domain SDK setups
	// (requires AppCookieSecure=true). Use "lax" or "strict" for same-origin deployments.
	AppCookieSameSite string
	// IsAdminCookieSecure is the flag to set secure(http only) cookie
	AdminCookieSecure bool
	// DisableAdminHeaderAuth is the flag to disable admin authentication via header
	DisableAdminHeaderAuth bool
	// Rate Limiting
	// RateLimitRPS is the maximum requests per second per IP
	RateLimitRPS int
	// RateLimitBurst is the maximum burst size per IP
	RateLimitBurst int
	// RateLimitFailClosed rejects requests when the rate limit backend errors (default: fail-open).
	RateLimitFailClosed bool
	// TrustedProxies is the list of CIDRs allowed to set X-Forwarded-For
	// and similar proxy headers. Empty (the default) means no proxies are
	// trusted and gin will use RemoteAddr directly. Operators behind a
	// reverse proxy MUST set this explicitly or rate limiting and audit
	// logs will key on the proxy IP, not the real client IP.
	TrustedProxies []string

	// BackchannelLogoutURI is the URL to which the server POSTs a
	// signed logout_token when a user logs out successfully. When
	// empty (default), back-channel logout notifications are disabled.
	// See OIDC Back-Channel Logout 1.0 §2.5 for the protocol.
	BackchannelLogoutURI string

	// OpenFGA / Fine-Grained Authorization engine. Authorizer embeds OpenFGA
	// in-process — it IS the engine. By default FGA reuses the main database
	// when it is OpenFGA-compatible (sqlite/postgres/mysql/mariadb); see
	// FGAStoreConfig. Fields below override that. When neither the main DB is
	// compatible nor FGAStore is set, the engine is not constructed and the
	// fga_* resolvers fail closed ("fine-grained authorization is not enabled").

	// FGAStore overrides the OpenFGA datastore kind: "sqlite", "postgres",
	// "mysql", or "memory" (dev/tests). Empty = reuse the main database when it
	// is SQL-compatible; required only for unsupported main DBs (mongodb,
	// dynamodb, …) or to use a dedicated FGA store.
	FGAStore string
	// FGAStoreURL is the connection URI for an overridden FGAStore: a file: URI
	// for sqlite, or a DSN for postgres/mysql. Ignored when FGA reuses the main
	// database or for the memory store.
	FGAStoreURL string
}

// Finalize derives runtime config from raw flag values. It must run after flags
// are parsed and before any provider is built (wired via the root command's
// PersistentPreRun so every subcommand, including `mcp`, is consistent). It is
// idempotent.
func (c *Config) Finalize() {
	// Provider availability is derived from credentials being present.
	c.IsEmailServiceEnabled = strings.TrimSpace(c.SMTPHost) != "" &&
		c.SMTPPort > 0 &&
		strings.TrimSpace(c.SMTPSenderEmail) != ""
	c.IsSMSServiceEnabled = strings.TrimSpace(c.TwilioAPIKey) != "" &&
		strings.TrimSpace(c.TwilioAPISecret) != "" &&
		strings.TrimSpace(c.TwilioAccountSID) != "" &&
		strings.TrimSpace(c.TwilioSender) != ""

	// MFA methods are on by default; operators opt out via --disable-*.
	c.EnableTOTPLogin = !c.DisableTOTPLogin
	c.EnableEmailOTP = !c.DisableEmailOTP
	c.EnableSMSOTP = !c.DisableSMSOTP
	c.EnableWebauthnMFA = !c.DisableWebauthnMFA

	// MFA is available when at least one method is usable. Email/SMS OTP need
	// their provider configured; TOTP has no external dependency. Deriving this
	// (rather than a standalone --enable-mfa flag) prevents the state where MFA
	// is "enabled" while every method is unavailable.
	c.EnableMFA = c.EnableTOTPLogin ||
		c.EnableWebauthnMFA ||
		(c.EnableEmailOTP && c.IsEmailServiceEnabled) ||
		(c.EnableSMSOTP && c.IsSMSServiceEnabled)

	// One-way global kill switch. Wins over everything: no MFA challenge is
	// possible and enforcement is neutralized so signup cannot flag users for
	// an MFA they can never complete. Does not affect WebAuthn/passkey as a
	// primary login method — only its role as an MFA factor.
	if c.DisableMFA {
		c.EnableMFA = false
		c.EnforceMFA = false
	}
}

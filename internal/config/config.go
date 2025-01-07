package config

// Config defines the configuration for the authorizer instance
type Config struct {
	// Env is the environment of the authorizer instance
	Env string
	// OrganizationLogo is the logo of the organization
	OrganizationLogo string
	// OrganizationName is the name of the organization
	OrganizationName string
	// AdminSecret is the secret for the admin
	AdminSecret string
	// AllowedOrigins is the list of allowed origins
	AllowedOrigins []string

	// DisableLoginPage is the flag to disable login page
	DisableLoginPage bool
	// DisablePlayground is the flag to disable playground
	DisablePlayground bool

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
	SkipTLSVerification bool

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
	// DisableStrongPassword is the flag to disable strong password
	DisableStrongPassword bool
	// DisableTOTPLogin boolean to disable TOTP login
	DisableTOTPLogin bool
	// DisableBasicAuthentication boolean to disable basic authentication
	DisableBasicAuthentication bool
	// DisableMagicLinkLogin boolean to disable magic link login
	DisableMagicLinkLogin bool
	// DisableEmailVerification boolean to disable email verification
	DisableEmailVerification bool
	// DisableMobileBasicAuthentication boolean to disable mobile basic authentication
	DisableMobileBasicAuthentication bool
	// DisablePhoneVerification boolean to disable phone verification
	DisablePhoneVerification bool
	// DisableMFA boolean to disable MFA
	DisableMFA bool
	// DisableEmailOTP boolean to disable email OTP
	DisableEmailOTP bool
	// DisableSMSOTP boolean to disable SMS OTP
	DisableSMSOTP bool
	// DisableSignup boolean to disable signup
	DisableSignup bool
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
	// JWTRoleClaim is the role claim for the JWT
	JWTRoleClaim string
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

	// GithubClientID is the client ID for Github OAuth
	GithubClientID string
	// GithubClientSecret is the client secret for Github OAuth
	GithubClientSecret string

	// FacebookClientID is the client ID for Facebook OAuth
	FacebookClientID string
	// FacebookClientSecret is the client secret for Facebook OAuth
	FacebookClientSecret string

	// LinkedinClientID is the client ID for Linkedin OAuth
	LinkedinClientID string
	// LinkedinClientSecret is the client secret for Linkedin OAuth
	LinkedinClientSecret string

	// TwitterClientID is the client ID for Twitter OAuth
	TwitterClientID string
	// TwitterClientSecret is the client secret for Twitter OAuth
	TwitterClientSecret string

	// MicrosoftClientID is the client ID for Microsoft OAuth
	MicrosoftClientID string
	// MicrosoftClientSecret is the client secret for Microsoft OAuth
	MicrosoftClientSecret string
	// MicrosoftTenantID is the tenant ID for Microsoft OAuth
	MicrosoftTenantID string

	// AppleClientID is the client ID for Apple OAuth
	AppleClientID string
	// AppleClientSecret is the client secret for Apple OAuth
	AppleClientSecret string

	// DiscordClientID is the client ID for Discord OAuth
	DiscordClientID string
	// DiscordClientSecret is the client secret for Discord OAuth
	DiscordClientSecret string

	// TwitchClientID is the client ID for Twitch OAuth
	TwitchClientID string
	// TwitchClientSecret is the client secret for Twitch OAuth
	TwitchClientSecret string

	// RoboloxClientID is the client ID for Robolox OAuth
	RoboloxClientID string
	// RoboloxClientSecret is the client secret for Robolox OAuth
	RoboloxClientSecret string
}

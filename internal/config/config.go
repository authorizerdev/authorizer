package config

// Config defines the configuration for the authorizer instance
type Config struct {
	// Env is the environment of the authorizer instance
	Env string
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

	// User Configurations
	// DefaultRoles is the default roles for the user
	// It is a comma separated string
	DefaultRoles string
}

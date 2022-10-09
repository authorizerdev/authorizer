package constants

var VERSION = "0.0.1"

const (
	// TestEnv is used for testing
	TestEnv = "test"
	// EnvKeyEnv key for env variable ENV
	EnvKeyEnv = "ENV"
	// EnvKeyEnvPath key for cli arg variable ENV_PATH
	EnvKeyEnvPath = "ENV_PATH"
	// EnvKeyAuthorizerURL key for env variable AUTHORIZER_URL
	EnvKeyAuthorizerURL = "AUTHORIZER_URL"
	// EnvKeyPort key for env variable PORT
	EnvKeyPort = "PORT"
	// EnvKeyAccessTokenExpiryTime key for env variable ACCESS_TOKEN_EXPIRY_TIME
	EnvKeyAccessTokenExpiryTime = "ACCESS_TOKEN_EXPIRY_TIME"
	// EnvKeyAdminSecret key for env variable ADMIN_SECRET
	EnvKeyAdminSecret = "ADMIN_SECRET"
	// EnvKeyDatabaseType key for env variable DATABASE_TYPE
	EnvKeyDatabaseType = "DATABASE_TYPE"
	// EnvKeyDatabaseURL key for env variable DATABASE_URL
	EnvKeyDatabaseURL = "DATABASE_URL"
	// EnvAwsRegion key for env variable AWS REGION
	EnvAwsRegion = "REGION"
	// EnvAwsAccessKey key for env variable AWS_ACCESS_KEY
	EnvAwsAccessKey = "AWS_ACCESS_KEY"
	// EnvAwsAccessKey key for env variable AWS_SECRET_KEY
	EnvAwsSecretKey = "AWS_SECRET_KEY"
	// EnvAwsAccessKey key for env variable AWS_SECRET_KEY
	EnvCouchbaseScope = "COUCHBASE_SCOPE"
	// EnvAwsAccessKey key for env variable AWS_SECRET_KEY
	EnvCouchbaseBucket = "COUCHBASE_BUCKET"
	// EnvKeyDatabaseName key for env variable DATABASE_NAME
	EnvKeyDatabaseName = "DATABASE_NAME"
	// EnvKeyDatabaseUsername key for env variable DATABASE_USERNAME
	EnvKeyDatabaseUsername = "DATABASE_USERNAME"
	// EnvKeyDatabasePassword key for env variable DATABASE_PASSWORD
	EnvKeyDatabasePassword = "DATABASE_PASSWORD"
	// EnvKeyDatabasePort key for env variable DATABASE_PORT
	EnvKeyDatabasePort = "DATABASE_PORT"
	// EnvKeyDatabaseHost key for env variable DATABASE_HOST
	EnvKeyDatabaseHost = "DATABASE_HOST"
	// EnvKeyDatabaseCert key for env variable DATABASE_CERT
	EnvKeyDatabaseCert = "DATABASE_CERT"
	// EnvKeyDatabaseCertKey key for env variable DATABASE_KEY
	EnvKeyDatabaseCertKey = "DATABASE_CERT_KEY"
	// EnvKeyDatabaseCACert key for env variable DATABASE_CA_CERT
	EnvKeyDatabaseCACert = "DATABASE_CA_CERT"
	// EnvKeySmtpHost key for env variable SMTP_HOST
	EnvKeySmtpHost = "SMTP_HOST"
	// EnvKeySmtpPort key for env variable SMTP_PORT
	EnvKeySmtpPort = "SMTP_PORT"
	// EnvKeySmtpUsername key for env variable SMTP_USERNAME
	EnvKeySmtpUsername = "SMTP_USERNAME"
	// EnvKeySmtpPassword key for env variable SMTP_PASSWORD
	EnvKeySmtpPassword = "SMTP_PASSWORD"
	// EnvKeySenderEmail key for env variable SENDER_EMAIL
	EnvKeySenderEmail = "SENDER_EMAIL"
	// EnvKeyIsEmailServiceEnabled key for env variable IS_EMAIL_SERVICE_ENABLED
	EnvKeyIsEmailServiceEnabled = "IS_EMAIL_SERVICE_ENABLED"
	// EnvKeyAppCookieSecure key for env variable APP_COOKIE_SECURE
	EnvKeyAppCookieSecure = "APP_COOKIE_SECURE"
	// EnvKeyAdminCookieSecure key for env variable ADMIN_COOKIE_SECURE
	EnvKeyAdminCookieSecure = "ADMIN_COOKIE_SECURE"
	// EnvKeyJwtType key for env variable JWT_TYPE
	EnvKeyJwtType = "JWT_TYPE"
	// EnvKeyJwtSecret key for env variable JWT_SECRET
	EnvKeyJwtSecret = "JWT_SECRET"
	// EnvKeyJwtPrivateKey key for env variable JWT_PRIVATE_KEY
	EnvKeyJwtPrivateKey = "JWT_PRIVATE_KEY"
	// EnvKeyJwtPublicKey key for env variable JWT_PUBLIC_KEY
	EnvKeyJwtPublicKey = "JWT_PUBLIC_KEY"
	// EnvKeyAppURL key for env variable APP_URL
	EnvKeyAppURL = "APP_URL"
	// EnvKeyRedisURL key for env variable REDIS_URL
	EnvKeyRedisURL = "REDIS_URL"
	// EnvKeyResetPasswordURL key for env variable RESET_PASSWORD_URL
	EnvKeyResetPasswordURL = "RESET_PASSWORD_URL"
	// EnvKeyJwtRoleClaim key for env variable JWT_ROLE_CLAIM
	EnvKeyJwtRoleClaim = "JWT_ROLE_CLAIM"
	// EnvKeyGoogleClientID key for env variable GOOGLE_CLIENT_ID
	EnvKeyGoogleClientID = "GOOGLE_CLIENT_ID"
	// EnvKeyGoogleClientSecret key for env variable GOOGLE_CLIENT_SECRET
	EnvKeyGoogleClientSecret = "GOOGLE_CLIENT_SECRET"
	// EnvKeyGithubClientID key for env variable GITHUB_CLIENT_ID
	EnvKeyGithubClientID = "GITHUB_CLIENT_ID"
	// EnvKeyGithubClientSecret key for env variable GITHUB_CLIENT_SECRET
	EnvKeyGithubClientSecret = "GITHUB_CLIENT_SECRET"
	// EnvKeyFacebookClientID key for env variable FACEBOOK_CLIENT_ID
	EnvKeyFacebookClientID = "FACEBOOK_CLIENT_ID"
	// EnvKeyFacebookClientSecret key for env variable FACEBOOK_CLIENT_SECRET
	EnvKeyFacebookClientSecret = "FACEBOOK_CLIENT_SECRET"
	// EnvKeyLinkedinClientID key for env variable LINKEDIN_CLIENT_ID
	EnvKeyLinkedInClientID = "LINKEDIN_CLIENT_ID"
	// EnvKeyLinkedinClientSecret key for env variable LINKEDIN_CLIENT_SECRET
	EnvKeyLinkedInClientSecret = "LINKEDIN_CLIENT_SECRET"
	// EnvKeyAppleClientID key for env variable APPLE_CLIENT_ID
	EnvKeyAppleClientID = "APPLE_CLIENT_ID"
	// EnvKeyAppleClientSecret key for env variable APPLE_CLIENT_SECRET
	EnvKeyAppleClientSecret = "APPLE_CLIENT_SECRET"
	// EnvKeyTwitterClientID key for env variable TWITTER_CLIENT_ID
	EnvKeyTwitterClientID = "TWITTER_CLIENT_ID"
	// EnvKeyTwitterClientSecret key for env variable TWITTER_CLIENT_SECRET
	EnvKeyTwitterClientSecret = "TWITTER_CLIENT_SECRET"
	// EnvKeyOrganizationName key for env variable ORGANIZATION_NAME
	EnvKeyOrganizationName = "ORGANIZATION_NAME"
	// EnvKeyOrganizationLogo key for env variable ORGANIZATION_LOGO
	EnvKeyOrganizationLogo = "ORGANIZATION_LOGO"
	// EnvKeyCustomAccessTokenScript key for env variable CUSTOM_ACCESS_TOKEN_SCRIPT
	EnvKeyCustomAccessTokenScript = "CUSTOM_ACCESS_TOKEN_SCRIPT"

	// Not Exposed Keys
	// EnvKeyClientID key for env variable CLIENT_ID
	EnvKeyClientID = "CLIENT_ID"
	// EnvKeyClientSecret key for env variable CLIENT_SECRET
	EnvKeyClientSecret = "CLIENT_SECRET"
	// EnvKeyEncryptionKey key for env variable ENCRYPTION_KEY
	EnvKeyEncryptionKey = "ENCRYPTION_KEY"
	// EnvKeyJWK key for env variable JWK
	EnvKeyJWK = "JWK"

	// Boolean variables
	// EnvKeyIsProd key for env variable IS_PROD
	EnvKeyIsProd = "IS_PROD"
	// EnvKeyDisableEmailVerification key for env variable DISABLE_EMAIL_VERIFICATION
	EnvKeyDisableEmailVerification = "DISABLE_EMAIL_VERIFICATION"
	// EnvKeyDisableBasicAuthentication key for env variable DISABLE_BASIC_AUTH
	EnvKeyDisableBasicAuthentication = "DISABLE_BASIC_AUTHENTICATION"
	// EnvKeyDisableMagicLinkLogin key for env variable DISABLE_MAGIC_LINK_LOGIN
	EnvKeyDisableMagicLinkLogin = "DISABLE_MAGIC_LINK_LOGIN"
	// EnvKeyDisableLoginPage key for env variable DISABLE_LOGIN_PAGE
	EnvKeyDisableLoginPage = "DISABLE_LOGIN_PAGE"
	// EnvKeyDisableSignUp key for env variable DISABLE_SIGN_UP
	EnvKeyDisableSignUp = "DISABLE_SIGN_UP"
	// EnvKeyDisableRedisForEnv key for env variable DISABLE_REDIS_FOR_ENV
	EnvKeyDisableRedisForEnv = "DISABLE_REDIS_FOR_ENV"
	// EnvKeyDisableStrongPassword key for env variable DISABLE_STRONG_PASSWORD
	EnvKeyDisableStrongPassword = "DISABLE_STRONG_PASSWORD"
	// EnvKeyEnforceMultiFactorAuthentication is key for env variable ENFORCE_MULTI_FACTOR_AUTHENTICATION
	// If enforced and changed later on, existing user will have MFA but new user will not have MFA
	EnvKeyEnforceMultiFactorAuthentication = "ENFORCE_MULTI_FACTOR_AUTHENTICATION"
	// EnvKeyDisableMultiFactorAuthentication is key for env variable DISABLE_MULTI_FACTOR_AUTHENTICATION
	// this variable is used to completely disable multi factor authentication. It will have no effect on profile preference
	EnvKeyDisableMultiFactorAuthentication = "DISABLE_MULTI_FACTOR_AUTHENTICATION"

	// Slice variables
	// EnvKeyRoles key for env variable ROLES
	EnvKeyRoles = "ROLES"
	// EnvKeyProtectedRoles key for env variable PROTECTED_ROLES
	EnvKeyProtectedRoles = "PROTECTED_ROLES"
	// EnvKeyDefaultRoles key for env variable DEFAULT_ROLES
	EnvKeyDefaultRoles = "DEFAULT_ROLES"
	// EnvKeyAllowedOrigins key for env variable ALLOWED_ORIGINS
	EnvKeyAllowedOrigins = "ALLOWED_ORIGINS"
)

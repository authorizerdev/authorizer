package constants

var (
	ADMIN_SECRET                 = ""
	ENV                          = ""
	VERSION                      = ""
	DATABASE_TYPE                = ""
	DATABASE_URL                 = ""
	SMTP_HOST                    = ""
	SMTP_PORT                    = ""
	SENDER_EMAIL                 = ""
	SENDER_PASSWORD              = ""
	JWT_TYPE                     = ""
	JWT_SECRET                   = ""
	ALLOWED_ORIGINS              = []string{}
	AUTHORIZER_URL               = ""
	APP_URL                      = ""
	PORT                         = "8080"
	REDIS_URL                    = ""
	IS_PROD                      = false
	COOKIE_NAME                  = ""
	RESET_PASSWORD_URL           = ""
	DISABLE_EMAIL_VERIFICATION   = "false"
	DISABLE_BASIC_AUTHENTICATION = "false"
	DISABLE_MAGIC_LOGIN          = "false"

	// ROLES
	ROLES           = []string{}
	PROTECTED_ROLES = []string{}
	DEFAULT_ROLES   = []string{}
	JWT_ROLE_CLAIM  = "role"

	// OAuth login
	GOOGLE_CLIENT_ID       = ""
	GOOGLE_CLIENT_SECRET   = ""
	GITHUB_CLIENT_ID       = ""
	GITHUB_CLIENT_SECRET   = ""
	FACEBOOK_CLIENT_ID     = ""
	FACEBOOK_CLIENT_SECRET = ""
	TWITTER_CLIENT_ID      = ""
	TWITTER_CLIENT_SECRET  = ""

	// Org envs
	ORGANIZATION_NAME = "Authorizer"
	ORGANIZATION_LOGO = "https://authorizer.dev/images/logo.png"
)

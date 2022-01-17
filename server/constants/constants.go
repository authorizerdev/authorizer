package constants

type EnvConst struct {
	ADMIN_SECRET                 string
	ENV                          string
	ENV_PATH                     string
	VERSION                      string
	DATABASE_TYPE                string
	DATABASE_URL                 string
	DATABASE_NAME                string
	SMTP_HOST                    string
	SMTP_PORT                    string
	SMTP_PASSWORD                string
	SMTP_USERNAME                string
	SENDER_EMAIL                 string
	JWT_TYPE                     string
	JWT_SECRET                   string
	ALLOWED_ORIGINS              []string
	AUTHORIZER_URL               string
	APP_URL                      string
	PORT                         string
	REDIS_URL                    string
	COOKIE_NAME                  string
	ADMIN_COOKIE_NAME            string
	RESET_PASSWORD_URL           string
	ENCRYPTION_KEY               string `json:"-"`
	IS_PROD                      bool
	DISABLE_EMAIL_VERIFICATION   bool
	DISABLE_BASIC_AUTHENTICATION bool
	DISABLE_MAGIC_LINK_LOGIN     bool
	DISABLE_LOGIN_PAGE           bool

	// ROLES
	ROLES           []string
	PROTECTED_ROLES []string
	DEFAULT_ROLES   []string
	JWT_ROLE_CLAIM  string

	// OAuth login
	GOOGLE_CLIENT_ID       string
	GOOGLE_CLIENT_SECRET   string
	GITHUB_CLIENT_ID       string
	GITHUB_CLIENT_SECRET   string
	FACEBOOK_CLIENT_ID     string
	FACEBOOK_CLIENT_SECRET string

	// Org envs
	ORGANIZATION_NAME string
	ORGANIZATION_LOGO string
}

var EnvData = EnvConst{
	ADMIN_COOKIE_NAME:            "authorizer-admin",
	JWT_ROLE_CLAIM:               "role",
	ORGANIZATION_NAME:            "Authorizer",
	ORGANIZATION_LOGO:            "https://authorizer.dev/images/logo.png",
	DISABLE_EMAIL_VERIFICATION:   false,
	DISABLE_BASIC_AUTHENTICATION: false,
	DISABLE_MAGIC_LINK_LOGIN:     false,
	DISABLE_LOGIN_PAGE:           false,
	IS_PROD:                      false,
}

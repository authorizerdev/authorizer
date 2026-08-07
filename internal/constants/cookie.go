package constants

const (
	// AppCookieName is the name of the cookie that is used to store the application token
	AppCookieName = "cookie"
	// AdminCookieName is the name of the cookie that is used to store the admin token
	AdminCookieName = "authorizer-admin"
	// OAuthStateCookieName is the name of the cookie binding an in-flight social
	// login to the browser that started it. See internal/cookie/oauth_state.go.
	OAuthStateCookieName = "authorizer-oauth-state"
	// MfaCookieName is the name of the cookie that is used to store the mfa session
	MfaCookieName = "mfa"
)

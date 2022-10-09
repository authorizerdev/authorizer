package constants

const (
	// - query: for Authorization Code grant. 302 Found triggers redirect.
	ResponseModeQuery = "query"
	// - fragment: for Implicit grant. 302 Found triggers redirect.
	ResponseModeFragment = "fragment"
	// - form_post: 200 OK with response parameters embedded in an HTML form as hidden parameters.
	ResponseModeFormPost = "form_post"
	// - web_message: For Silent Authentication. Uses HTML5 web messaging.
	ResponseModeWebMessage = "web_message"

	// For the Authorization Code grant, use response_type=code to include the authorization code.
	ResponseTypeCode = "code"
	// For the Implicit grant, use response_type=token to include an access token.
	ResponseTypeToken = "token"
)

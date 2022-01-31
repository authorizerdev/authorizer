package cookie

import (
	"net/http"
	"net/url"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/envstore"
	"github.com/authorizerdev/authorizer/server/utils"
	"github.com/gin-gonic/gin"
)

// SetCookie sets the cookie in the response. It sets 4 cookies
// 1 COOKIE_NAME.access_token jwt token for the host (temp.abc.com)
// 2 COOKIE_NAME.access_token.domain jwt token for the domain (abc.com).
// 3 COOKIE_NAME.fingerprint fingerprint hash for the refresh token verification.
// 4 COOKIE_NAME.refresh_token refresh token
// Note all sites don't allow 2nd type of cookie
func SetCookie(gc *gin.Context, accessToken, refreshToken, fingerprintHash string) {
	secure := true
	httpOnly := true
	hostname := utils.GetHost(gc)
	host, _ := utils.GetHostParts(hostname)
	domain := utils.GetDomainName(hostname)
	if domain != "localhost" {
		domain = "." + domain
	}

	year := 60 * 60 * 24 * 365
	thirtyMin := 60 * 30

	gc.SetSameSite(http.SameSiteNoneMode)
	// set cookie for host
	gc.SetCookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".access_token", accessToken, thirtyMin, "/", host, secure, httpOnly)

	// in case of subdomain, set cookie for domain
	gc.SetCookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".access_token.domain", accessToken, thirtyMin, "/", domain, secure, httpOnly)

	// set finger print cookie (this should be accessed via cookie only)
	gc.SetCookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".fingerprint", fingerprintHash, year, "/", host, secure, httpOnly)

	// set refresh token cookie (this should be accessed via cookie only)
	gc.SetCookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".refresh_token", refreshToken, year, "/", host, secure, httpOnly)
}

// GetAccessTokenCookie to get access token cookie from the request
func GetAccessTokenCookie(gc *gin.Context) (string, error) {
	cookie, err := gc.Request.Cookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName) + ".access_token")
	if err != nil {
		cookie, err = gc.Request.Cookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName) + ".access_token.domain")
		if err != nil {
			return "", err
		}
	}

	return cookie.Value, nil
}

// GetRefreshTokenCookie to get refresh token cookie
func GetRefreshTokenCookie(gc *gin.Context) (string, error) {
	cookie, err := gc.Request.Cookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName) + ".refresh_token")
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

// GetFingerPrintCookie to get fingerprint cookie
func GetFingerPrintCookie(gc *gin.Context) (string, error) {
	cookie, err := gc.Request.Cookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName) + ".fingerprint")
	if err != nil {
		return "", err
	}

	// cookie escapes special characters like $
	// hence we need to unescape before comparing
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return "", err
	}

	return decodedValue, nil
}

// DeleteCookie sets response cookies to expire
func DeleteCookie(gc *gin.Context) {
	secure := true
	httpOnly := true
	hostname := utils.GetHost(gc)
	host, _ := utils.GetHostParts(hostname)
	domain := utils.GetDomainName(hostname)
	if domain != "localhost" {
		domain = "." + domain
	}

	gc.SetSameSite(http.SameSiteNoneMode)
	gc.SetCookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".access_token", "", -1, "/", host, secure, httpOnly)
	gc.SetCookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".access_token.domain", "", -1, "/", domain, secure, httpOnly)
	gc.SetCookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".fingerprint", "", -1, "/", host, secure, httpOnly)
	gc.SetCookie(envstore.EnvInMemoryStoreObj.GetStringStoreEnvVariable(constants.EnvKeyCookieName)+".refresh_token", "", -1, "/", host, secure, httpOnly)
}

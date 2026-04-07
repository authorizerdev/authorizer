package http_handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// userInfoProfileClaims is the set of claim names that must be returned
// when the access token includes the "profile" scope per OIDC Core §5.4.
var userInfoProfileClaims = map[string]struct{}{
	"name":               {},
	"family_name":        {},
	"given_name":         {},
	"middle_name":        {},
	"nickname":           {},
	"preferred_username": {},
	"profile":            {},
	"picture":            {},
	"website":            {},
	"gender":             {},
	"birthdate":          {},
	"zoneinfo":           {},
	"locale":             {},
	"updated_at":         {},
}

// userInfoEmailClaims is the set of claim names that must be returned
// when the access token includes the "email" scope.
var userInfoEmailClaims = map[string]struct{}{
	"email":          {},
	"email_verified": {},
}

// userInfoPhoneClaims is the set of claim names that must be returned
// when the access token includes the "phone" scope.
var userInfoPhoneClaims = map[string]struct{}{
	"phone_number":          {},
	"phone_number_verified": {},
}

// userInfoAddressClaims is the set of claim names that must be returned
// when the access token includes the "address" scope.
var userInfoAddressClaims = map[string]struct{}{
	"address": {},
}

// extractScopesFromAccessToken returns the lowercase scope set encoded in
// the access token. It accepts both the spec form (string-array claim) and
// the OAuth 2.0 RFC 6749 string form ("openid profile email").
func extractScopesFromAccessToken(claims map[string]interface{}) map[string]struct{} {
	out := map[string]struct{}{}
	if claims == nil {
		return out
	}
	switch v := claims["scope"].(type) {
	case string:
		for _, s := range strings.Fields(v) {
			out[strings.ToLower(s)] = struct{}{}
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				out[strings.ToLower(s)] = struct{}{}
			}
		}
	case []string:
		for _, s := range v {
			out[strings.ToLower(s)] = struct{}{}
		}
	}
	return out
}

// filterUserInfoByScopes implements OIDC Core §5.4: only the claims
// permitted by the requested scope groups are returned. The "sub" claim
// is always returned. Standard scope groups are profile, email, phone,
// address. Custom or unknown claims are not returned in strict mode.
func filterUserInfoByScopes(full map[string]interface{}, scopes map[string]struct{}) map[string]interface{} {
	filtered := map[string]interface{}{
		"sub": full["sub"],
	}
	// allow copies every claim key in the requested group into the filtered
	// response. Per OIDC Core §5.4 the keys associated with a granted scope
	// are part of the response shape; if the underlying user object has no
	// value for a claim we still emit the key with a JSON null so callers
	// can rely on a stable schema.
	allow := func(set map[string]struct{}) {
		for k := range set {
			if v, ok := full[k]; ok {
				filtered[k] = v
			} else {
				filtered[k] = nil
			}
		}
	}
	if _, ok := scopes["profile"]; ok {
		allow(userInfoProfileClaims)
	}
	if _, ok := scopes["email"]; ok {
		allow(userInfoEmailClaims)
	}
	if _, ok := scopes["phone"]; ok {
		allow(userInfoPhoneClaims)
	}
	if _, ok := scopes["address"]; ok {
		allow(userInfoAddressClaims)
	}
	return filtered
}

func (h *httpProvider) UserInfoHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "UserInfoHandler").Logger()
	return func(gc *gin.Context) {
		accessToken, err := h.TokenProvider.GetAccessToken(gc)
		if err != nil {
			log.Debug().Msg("Error getting access token")
			// RFC 6750 §3: No credentials - return 401 with WWW-Authenticate
			gc.Header("WWW-Authenticate", `Bearer realm="authorizer"`)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_request",
				"error_description": "No access token provided",
			})
			return
		}
		claims, err := h.TokenProvider.ValidateAccessToken(gc, accessToken)
		if err != nil {
			log.Debug().Msg("Error validating access token")
			// RFC 6750 §3.1: Invalid token - return 401 with error details
			gc.Header("WWW-Authenticate", `Bearer realm="authorizer", error="invalid_token", error_description="The access token is invalid or has expired"`)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "The access token is invalid or has expired",
			})
			return
		}
		userID, _ := claims["sub"].(string)
		user, err := h.StorageProvider.GetUserByID(gc, userID)
		if err != nil {
			log.Debug().Msg("Error getting user by ID")
			gc.Header("WWW-Authenticate", `Bearer realm="authorizer", error="invalid_token", error_description="The user associated with this token was not found"`)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "The user associated with this token was not found",
			})
			return
		}
		apiUser := user.AsAPIUser()
		userBytes, err := json.Marshal(apiUser)
		if err != nil {
			log.Debug().Msg("Error marshalling user")
			gc.JSON(http.StatusInternalServerError, gin.H{
				"error":             "server_error",
				"error_description": "Failed to process user data",
			})
			return
		}
		full := map[string]interface{}{}
		if err := json.Unmarshal(userBytes, &full); err != nil {
			log.Debug().Msg("Error unmarshalling user")
			gc.JSON(http.StatusInternalServerError, gin.H{
				"error":             "server_error",
				"error_description": "Failed to process user data",
			})
			return
		}
		// OIDC Core §5.3.2: sub claim MUST always be returned.
		full["sub"] = userID

		if h.Config.OIDCStrictUserInfoScopes {
			scopes := extractScopesFromAccessToken(claims)
			gc.JSON(http.StatusOK, filterUserInfoByScopes(full, scopes))
			return
		}

		// Backward-compatible lenient mode: return everything we have.
		gc.JSON(http.StatusOK, full)
	}
}

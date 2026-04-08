package http_handlers

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/parsers"
)

// equalConstantTime returns true when a and b are byte-equal in
// constant time. Used for client_id / client_secret comparison to
// prevent timing oracles per the project's security rules.
func equalConstantTime(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// IntrospectHandler implements RFC 7662 OAuth 2.0 Token Introspection at
// POST /oauth/introspect. Accepts application/x-www-form-urlencoded bodies
// and both HTTP Basic client authentication (client_secret_basic) and
// form-body client authentication (client_secret_post). Always returns
// {"active": false} for any inactive, invalid, or unknown token per
// RFC 7662 §2.2 — never leak details about why a token is inactive.
func (h *httpProvider) IntrospectHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "IntrospectHandler").Logger()
	return func(gc *gin.Context) {
		// RFC 7662 §2.2 + standard OAuth 2.0 cache discipline.
		gc.Writer.Header().Set("Cache-Control", "no-store")
		gc.Writer.Header().Set("Pragma", "no-cache")

		// Parse form body. Per RFC 7662 §2.1, the optional
		// token_type_hint is intentionally ignored: this server validates
		// any presented JWT against issuer/audience/expiry uniformly, so
		// the hint provides no useful disambiguation.
		tokenValue := strings.TrimSpace(gc.PostForm("token"))

		clientID := strings.TrimSpace(gc.PostForm("client_id"))
		clientSecret := strings.TrimSpace(gc.PostForm("client_secret"))

		// If no form creds, fall back to HTTP Basic.
		hasBasicAuth := false
		if clientID == "" && clientSecret == "" {
			if id, secret, ok := gc.Request.BasicAuth(); ok {
				clientID = id
				clientSecret = secret
				hasBasicAuth = true
			}
		}

		if clientID == "" {
			log.Debug().Msg("client_id missing on introspect request")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "The client_id parameter is required",
			})
			return
		}

		// RFC 7662 §2.1 + RFC 6749 §2.3: client authentication is
		// MANDATORY on the introspection endpoint. Validate client_id
		// (constant-time) and, when the server has a client_secret
		// configured, require a matching client_secret. Empty/missing
		// client_secret MUST be rejected — otherwise a caller could
		// authenticate with client_id alone and bypass authentication.
		clientFailed := false
		if !equalConstantTime(h.Config.ClientID, clientID) {
			log.Debug().Str("client_id", clientID).Msg("client_id mismatch on introspect")
			clientFailed = true
		} else if h.Config.ClientSecret != "" {
			// A secret is configured: it must be supplied AND match.
			if clientSecret == "" || !equalConstantTime(h.Config.ClientSecret, clientSecret) {
				log.Debug().Msg("client_secret missing or mismatched on introspect")
				clientFailed = true
			}
		}
		if clientFailed {
			// RFC 6749 §5.2: client auth failures via Basic return 401
			// with WWW-Authenticate; failures via form-post return 401
			// without the challenge header (or 400 if no auth scheme is
			// indicated). We return 401 in both cases to make the
			// authentication failure unambiguous.
			if hasBasicAuth {
				gc.Header("WWW-Authenticate", `Basic realm="introspect"`)
			}
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_client",
				"error_description": "Client authentication failed",
			})
			return
		}

		if tokenValue == "" {
			log.Debug().Msg("token parameter missing")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "The token parameter is required",
			})
			return
		}

		// Parse the token. Any failure → inactive.
		claims, err := h.TokenProvider.ParseJWTToken(tokenValue)
		if err != nil || claims == nil {
			gc.JSON(http.StatusOK, gin.H{"active": false})
			return
		}

		// Check exp. ParseJWTToken normalizes exp/iat to int64, but tolerate
		// other numeric encodings defensively.
		now := time.Now().Unix()
		if expRaw, ok := claims["exp"]; ok {
			var exp int64
			switch v := expRaw.(type) {
			case int64:
				exp = v
			case float64:
				exp = int64(v)
			case int:
				exp = int64(v)
			}
			if exp <= now {
				gc.JSON(http.StatusOK, gin.H{"active": false})
				return
			}
		} else {
			// Missing exp — treat as inactive; OIDC/OAuth tokens must have exp.
			gc.JSON(http.StatusOK, gin.H{"active": false})
			return
		}

		// Check iss matches this server.
		hostname := parsers.GetHost(gc)
		if issClaim, _ := claims["iss"].(string); issClaim == "" || issClaim != hostname {
			gc.JSON(http.StatusOK, gin.H{"active": false})
			return
		}

		// Check aud matches our client_id.
		if !audienceMatchesIntrospect(claims["aud"], h.Config.ClientID) {
			gc.JSON(http.StatusOK, gin.H{"active": false})
			return
		}

		// Build active response. Omit keys whose source value is missing.
		resp := gin.H{"active": true}
		copyIfPresent := func(srcKey, dstKey string) {
			if v, ok := claims[srcKey]; ok && v != nil && v != "" {
				resp[dstKey] = v
			}
		}
		copyIfPresent("scope", "scope")
		// RFC 7662 §2.2: client_id MUST be a string — set it directly
		// from h.Config.ClientID rather than copying from the `aud`
		// claim, which may be a JSON array for multi-audience tokens.
		// The audience check above already confirmed h.Config.ClientID
		// is in the audience set.
		resp["client_id"] = h.Config.ClientID
		copyIfPresent("exp", "exp")
		copyIfPresent("iat", "iat")
		copyIfPresent("sub", "sub")
		copyIfPresent("aud", "aud")
		copyIfPresent("iss", "iss")
		if tt, ok := claims["token_type"].(string); ok && tt != "" {
			// RFC 7662: pass through the token_type claim value (e.g.
			// "access_token", "refresh_token", "id_token") as recorded
			// at issuance.
			resp["token_type"] = tt
		}
		gc.JSON(http.StatusOK, resp)
	}
}

// audienceMatchesIntrospect accepts either a string aud or a []interface{}
// aud claim and returns true if it contains the expected client ID.
func audienceMatchesIntrospect(audClaim interface{}, expected string) bool {
	if expected == "" {
		return false
	}
	switch v := audClaim.(type) {
	case string:
		return v == expected
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == expected {
				return true
			}
		}
	case []string:
		for _, s := range v {
			if s == expected {
				return true
			}
		}
	}
	return false
}

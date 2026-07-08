package http_handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/service/clientauth"
)

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

		// Parse form body.
		tokenValue := strings.TrimSpace(gc.PostForm("token"))
		tokenTypeHint := strings.TrimSpace(gc.PostForm("token_type_hint"))
		_ = tokenTypeHint // Per RFC 7662 §2.1, unknown hints are ignored.

		// RFC 7662 §2.1: the introspection caller MUST authenticate. Route the
		// credential through the shared registry resolver (RFC 6749 §2.3) so the
		// caller is authenticated against the authoritative client registry rather
		// than the single static Config.ClientID/Secret. RequireSecret enforces a
		// secret on every caller (introspection is a confidential-client endpoint);
		// the reserved client keeps working via the resolver's Config fallback when
		// its registry row is absent.
		basicClientID, basicClientSecret, hasBasicAuth := gc.Request.BasicAuth()
		resolvedClient, authErr := h.clientAuthProvider.ResolveClient(gc, clientauth.ResolveParams{
			BodyClientID:  strings.TrimSpace(gc.PostForm("client_id")),
			BodySecret:    gc.PostForm("client_secret"),
			BasicClientID: basicClientID,
			BasicSecret:   basicClientSecret,
			HasBasicAuth:  hasBasicAuth,
			RequireSecret: true,
		})
		if authErr != nil {
			log.Debug().Err(authErr).Msg("introspection caller authentication failed")
			// Non-basic invalid_client maps to 400 on the introspection endpoint,
			// preserving the pre-registry response shape.
			respondResourceClientAuthError(gc, authErr, hasBasicAuth, http.StatusBadRequest)
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

		// Registry-aware audience check: a token's aud is its client's client_id.
		// The token is only "active" for the authenticated caller when it was
		// issued to that caller — validate against the resolved caller's client_id,
		// not a single static Config.ClientID. A token minted for a different client
		// yields {"active": false} (no oracle) rather than leaking its claims.
		if !audienceMatchesIntrospect(claims["aud"], resolvedClient.ClientID) {
			gc.JSON(http.StatusOK, gin.H{"active": false})
			return
		}

		// Revocation awareness: for a first-party user token (sub == user id), a
		// revoked/deprovisioned user's token must introspect as inactive even
		// before the short access-token TTL elapses (SCIM active:false, account
		// deactivation). RevokedTimestamp is the reliable, provider-agnostic
		// signal. A sub that resolves to no user (e.g. a machine/service-account
		// token) is left untouched — this check only demotes, never promotes.
		if sub, _ := claims["sub"].(string); sub != "" {
			if user, uErr := h.StorageProvider.GetUserByID(gc, sub); uErr == nil && user != nil && user.RevokedTimestamp != nil {
				gc.JSON(http.StatusOK, gin.H{"active": false})
				return
			}
		}

		// Build active response. Omit keys whose source value is missing.
		resp := gin.H{"active": true}
		copyIfPresent := func(srcKey, dstKey string) {
			if v, ok := claims[srcKey]; ok && v != nil && v != "" {
				resp[dstKey] = v
			}
		}
		copyIfPresent("scope", "scope")
		// RFC 7662 §2.2: client_id MUST be a string — set it to the resolved
		// caller's client_id rather than copying from the `aud` claim, which may be
		// a JSON array for multi-audience tokens. The audience check above already
		// confirmed the resolved client's id is in the audience set.
		resp["client_id"] = resolvedClient.ClientID
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

// respondResourceClientAuthError maps a clientauth resolver error for the
// resource-facing endpoints that authenticate a calling client — introspection
// (RFC 7662 §2.1) and revocation (RFC 7009 §2.1). A missing client_id or more
// than one auth method maps to invalid_request (400). Any other failure maps to
// invalid_client: 401 + WWW-Authenticate when the caller used HTTP Basic
// (RFC 6749 §5.2), otherwise nonBasicStatus (400 for introspection, 401 for
// revocation — each preserving that endpoint's pre-registry response).
func respondResourceClientAuthError(gc *gin.Context, err error, hasBasicAuth bool, nonBasicStatus int) {
	switch {
	case errors.Is(err, clientauth.ErrMissingClientID):
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "The client_id parameter is required",
		})
		return
	case errors.Is(err, clientauth.ErrMultipleAuthMethods):
		gc.JSON(http.StatusBadRequest, gin.H{
			"error":             "invalid_request",
			"error_description": "Only one client authentication method may be used per request",
		})
		return
	}
	if hasBasicAuth {
		gc.Header("WWW-Authenticate", `Basic realm="authorizer"`)
		gc.JSON(http.StatusUnauthorized, gin.H{
			"error":             "invalid_client",
			"error_description": "Client authentication failed",
		})
		return
	}
	gc.JSON(nonBasicStatus, gin.H{
		"error":             "invalid_client",
		"error_description": "Client authentication failed",
	})
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

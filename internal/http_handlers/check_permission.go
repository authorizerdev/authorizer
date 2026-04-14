package http_handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/authorization"
)

// checkPermissionRequest is the JSON body for POST /api/v1/check-permission.
type checkPermissionRequest struct {
	Resource string `json:"resource"`
	Scope    string `json:"scope"`
}

// checkPermissionResponse is the JSON response for POST /api/v1/check-permission.
type checkPermissionResponse struct {
	Allowed       bool   `json:"allowed"`
	MatchedPolicy string `json:"matched_policy"`
}

// CheckPermissionHandler handles POST /api/v1/check-permission for downstream
// services that need to verify whether a user has a specific permission without
// using GraphQL. It validates the Bearer token, builds a Principal from the
// token claims, and delegates to the authorization provider.
func (h *httpProvider) CheckPermissionHandler() gin.HandlerFunc {
	log := h.Log.With().Str("func", "CheckPermissionHandler").Logger()
	return func(gc *gin.Context) {
		// Extract and validate the access token from the Authorization header.
		accessToken, err := h.TokenProvider.GetAccessToken(gc)
		if err != nil {
			log.Debug().Msg("Missing or malformed access token")
			gc.Header("WWW-Authenticate", `Bearer realm="authorizer"`)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_request",
				"error_description": "No access token provided",
			})
			return
		}

		claims, err := h.TokenProvider.ValidateAccessToken(gc, accessToken)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid access token")
			gc.Header("WWW-Authenticate", `Bearer realm="authorizer", error="invalid_token", error_description="The access token is invalid or has expired"`)
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "The access token is invalid or has expired",
			})
			return
		}

		// Parse the request body.
		var req checkPermissionRequest
		if err := gc.ShouldBindJSON(&req); err != nil {
			log.Debug().Err(err).Msg("Invalid request body")
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "Request body must be JSON with 'resource' and 'scope' fields",
			})
			return
		}

		req.Resource = strings.TrimSpace(req.Resource)
		req.Scope = strings.TrimSpace(req.Scope)

		if req.Resource == "" || req.Scope == "" {
			gc.JSON(http.StatusBadRequest, gin.H{
				"error":             "invalid_request",
				"error_description": "Both 'resource' and 'scope' fields are required and must be non-empty",
			})
			return
		}

		// Build a Principal from the token claims.
		userID, _ := claims["sub"].(string)
		if userID == "" {
			gc.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "Token is missing a valid 'sub' claim",
			})
			return
		}

		var roles []string
		if rolesVal, ok := claims["roles"].([]interface{}); ok {
			for _, r := range rolesVal {
				if s, ok := r.(string); ok {
					roles = append(roles, s)
				}
			}
		}

		principal := &authorization.Principal{
			ID:    userID,
			Type:  "user",
			Roles: roles,
		}

		result, err := h.AuthorizationProvider.CheckPermission(gc.Request.Context(), principal, req.Resource, req.Scope)
		if err != nil {
			log.Error().Err(err).Msg("Authorization check failed")
			gc.JSON(http.StatusInternalServerError, gin.H{
				"error":             "server_error",
				"error_description": "Failed to evaluate permission",
			})
			return
		}

		gc.JSON(http.StatusOK, checkPermissionResponse{
			Allowed:       result.Allowed,
			MatchedPolicy: result.MatchedPolicy,
		})
	}
}

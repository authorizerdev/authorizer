// Package scim exposes the per-organization inbound SCIM 2.0 HTTP surface
// (RFC 7644) at /scim/v2/. Transport only: every operation delegates to
// internal/service/scim, and the org is resolved solely from the bearer token
// (design §4.4 H6). Errors use the SCIM error schema, not the OAuth envelope.
package scim

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	svcscim "github.com/authorizerdev/authorizer/internal/service/scim"
)

const (
	// contentType is the SCIM media type (RFC 7644 §3.1).
	contentType = "application/scim+json"

	schemaUser     = "urn:ietf:params:scim:schemas:core:2.0:User"
	schemaGroup    = "urn:ietf:params:scim:schemas:core:2.0:Group"
	schemaError    = "urn:ietf:params:scim:api:messages:2.0:Error"
	schemaListResp = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	schemaSPConfig = "urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"
	schemaRestype  = "urn:ietf:params:scim:schemas:core:2.0:ResourceType"
	schemaSchema   = "urn:ietf:params:scim:schemas:core:2.0:Schema"

	// ctxOrgID is the gin context key holding the org resolved from the token.
	ctxOrgID = "scim_org_id"
)

// Dependencies for the SCIM HTTP handler.
type Dependencies struct {
	Log     *zerolog.Logger
	Service svcscim.Provider
}

// Handler mounts the SCIM routes and authenticates every request via bearer
// token.
type Handler struct {
	Dependencies
}

// New constructs a SCIM HTTP handler.
func New(deps *Dependencies) *Handler {
	return &Handler{Dependencies: *deps}
}

// Register wires the SCIM routes onto a router group (mounted at /scim/v2). All
// routes are behind the bearer-auth middleware.
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.Use(h.authMiddleware())

	rg.POST("/Users", h.createUser)
	rg.GET("/Users", h.listUsers)
	rg.GET("/Users/:id", h.getUser)
	rg.PUT("/Users/:id", h.replaceUser)
	rg.PATCH("/Users/:id", h.patchUser)
	rg.DELETE("/Users/:id", h.deleteUser)

	rg.POST("/Groups", h.createGroup)
	rg.GET("/Groups", h.listGroups)
	rg.GET("/Groups/:id", h.getGroup)
	rg.PUT("/Groups/:id", h.replaceGroup)
	rg.PATCH("/Groups/:id", h.patchGroup)
	rg.DELETE("/Groups/:id", h.deleteGroup)

	rg.GET("/ServiceProviderConfig", h.serviceProviderConfig)
	rg.GET("/ResourceTypes", h.resourceTypes)
	rg.GET("/Schemas", h.schemas)
}

// scimError is the RFC 7644 §3.12 error body.
type scimError struct {
	Schemas  []string `json:"schemas"`
	Status   string   `json:"status"`
	SCIMType string   `json:"scimType,omitempty"`
	Detail   string   `json:"detail,omitempty"`
}

// writeError emits a SCIM-shaped error with the given HTTP status.
func writeError(c *gin.Context, status int, scimType, detail string) {
	c.Header("Content-Type", contentType)
	c.JSON(status, scimError{
		Schemas:  []string{schemaError},
		Status:   strconv.Itoa(status),
		SCIMType: scimType,
		Detail:   detail,
	})
	c.Abort()
}

// authMiddleware extracts the bearer token, resolves the org from it, and
// stores the org id in the gin context. A missing/invalid/disabled credential
// is a constant-time 401 — the org is NEVER taken from the URL or body (H6).
func (h *Handler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		bearer := ""
		if auth := c.GetHeader("Authorization"); len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
			bearer = auth[7:]
		}
		orgID, err := h.Service.Authenticate(c.Request.Context(), bearer)
		if err != nil {
			// Do not leak whether the token was absent, malformed, unknown, or
			// wrong — all collapse to one constant-time 401.
			writeError(c, http.StatusUnauthorized, "", "authentication failed")
			return
		}
		c.Set(ctxOrgID, orgID)
		c.Next()
	}
}

func (h *Handler) orgID(c *gin.Context) string {
	v, _ := c.Get(ctxOrgID)
	orgID, _ := v.(string)
	return orgID
}

// mapServiceError translates a scim-service sentinel into a SCIM HTTP error.
func mapServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, svcscim.ErrNotFound):
		writeError(c, http.StatusNotFound, "", "resource not found")
	case errors.Is(err, svcscim.ErrConflict):
		writeError(c, http.StatusConflict, "uniqueness", "userName already exists")
	case errors.Is(err, svcscim.ErrGroupConflict):
		writeError(c, http.StatusConflict, "uniqueness", "a group with this displayName already exists")
	case errors.Is(err, svcscim.ErrInvalid):
		writeError(c, http.StatusBadRequest, "invalidValue", "invalid request")
	case errors.Is(err, svcscim.ErrUnauthorized):
		writeError(c, http.StatusUnauthorized, "", "authentication failed")
	case errors.Is(err, svcscim.ErrGroupsUnavailable):
		writeError(c, http.StatusNotImplemented, "", "group provisioning is not enabled on this server")
	default:
		writeError(c, http.StatusInternalServerError, "", "internal error")
	}
}

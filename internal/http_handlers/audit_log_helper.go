package http_handlers

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// AuditLogOpts contains optional fields for an audit log entry.
type AuditLogOpts struct {
	ActorID      string
	ActorType    string // "user" or "admin"
	ActorEmail   string
	ResourceType string
	ResourceID   string
	Metadata     string
}

// logAuditEvent asynchronously records an audit log entry.
// It extracts request info (IP, UserAgent) from the Gin context before
// spawning the goroutine to avoid use-after-free on the request.
// Errors are logged but not propagated (fire-and-forget).
func (h *httpProvider) logAuditEvent(gc *gin.Context, action string, opts AuditLogOpts) {
	ipAddress := utils.GetIP(gc.Request)
	userAgent := utils.GetUserAgent(gc.Request)

	go func() {
		log := h.Log.With().Str("func", "logAuditEvent").Logger()
		auditLog := &schemas.AuditLog{
			ActorID:      opts.ActorID,
			ActorType:    opts.ActorType,
			ActorEmail:   opts.ActorEmail,
			Action:       action,
			ResourceType: opts.ResourceType,
			ResourceID:   opts.ResourceID,
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
			Metadata:     opts.Metadata,
		}
		if err := h.StorageProvider.AddAuditLog(context.Background(), auditLog); err != nil {
			log.Debug().Err(err).Str("action", action).Msg("Failed to add audit log")
		}
	}()
}

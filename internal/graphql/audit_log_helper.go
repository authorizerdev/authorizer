package graphql

import (
	"context"

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
// It extracts request info (IP, UserAgent) before spawning the goroutine
// to avoid accessing the Gin context after the request completes.
// Errors are logged but not propagated (fire-and-forget).
func (g *graphqlProvider) logAuditEvent(ctx context.Context, action string, opts AuditLogOpts) {
	log := g.Log.With().Str("func", "logAuditEvent").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext for audit log")
		return
	}
	ipAddress := utils.GetIP(gc.Request)
	userAgent := utils.GetUserAgent(gc.Request)

	go func() {
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
		if err := g.StorageProvider.AddAuditLog(context.Background(), auditLog); err != nil {
			log.Debug().Err(err).Str("action", action).Msg("Failed to add audit log")
		}
	}()
}

package audit

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies for the audit provider.
type Dependencies struct {
	Log             *zerolog.Logger
	StorageProvider storage.Provider
}

// Event represents an audit event to be logged.
type Event struct {
	ActorID      string
	ActorType    string
	ActorEmail   string
	Action       string
	ResourceType string
	ResourceID   string
	IPAddress    string
	UserAgent    string
	Metadata     string
}

// Provider is the interface for audit logging.
type Provider interface {
	// LogEvent asynchronously records an audit log entry.
	// It is fire-and-forget: errors are logged but not propagated.
	LogEvent(event Event)
}

type provider struct {
	deps *Dependencies
}

// Ensure provider implements Provider.
var _ Provider = &provider{}

// New creates a new audit provider.
func New(deps *Dependencies) Provider {
	return &provider{deps: deps}
}

// LogEvent asynchronously records an audit log entry.
func (p *provider) LogEvent(event Event) {
	go func() {
		log := p.deps.Log.With().Str("func", "LogEvent").Logger()
		auditLog := &schemas.AuditLog{
			ActorID:      event.ActorID,
			ActorType:    event.ActorType,
			ActorEmail:   event.ActorEmail,
			Action:       event.Action,
			ResourceType: event.ResourceType,
			ResourceID:   event.ResourceID,
			IPAddress:    event.IPAddress,
			UserAgent:    event.UserAgent,
			Metadata:     event.Metadata,
		}
		if err := p.deps.StorageProvider.AddAuditLog(context.Background(), auditLog); err != nil {
			log.Debug().Err(err).Str("action", event.Action).Msg("Failed to add audit log")
		}
	}()
}

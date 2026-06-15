package audit

import (
	"context"
	"encoding/json"

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
	// Protocol is the transport the operation came in on
	// (constants.Protocol{GraphQL,GRPC,REST}). It is folded into the persisted
	// Metadata JSON (under the "protocol" key) by LogEvent, so no audit-log
	// schema change is required. Empty when the caller did not set it.
	Protocol string
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

// metadataWithProtocol folds the transport protocol into the audit log's
// free-form Metadata column so the protocol is queryable without an audit-log
// schema change. When protocol is empty the metadata is returned unchanged.
// Otherwise: an empty metadata becomes {"protocol":"..."}; a metadata that is
// already a JSON object gains a "protocol" key; any other (non-JSON) metadata is
// preserved under a "metadata" key alongside "protocol".
func metadataWithProtocol(meta, protocol string) string {
	if protocol == "" {
		return meta
	}
	out := map[string]any{"protocol": protocol}
	if meta != "" {
		var existing map[string]any
		if json.Unmarshal([]byte(meta), &existing) == nil {
			for k, v := range existing {
				if k != "protocol" {
					out[k] = v
				}
			}
		} else {
			out["metadata"] = meta
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return meta
	}
	return string(b)
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
			Metadata:     metadataWithProtocol(event.Metadata, event.Protocol),
		}
		if err := p.deps.StorageProvider.AddAuditLog(context.Background(), auditLog); err != nil {
			log.Debug().Err(err).Str("action", event.Action).Msg("Failed to add audit log")
		}
	}()
}

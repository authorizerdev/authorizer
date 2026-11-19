package audit

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/asyncutil"
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

	// LogEventSync records an audit log entry synchronously and returns the
	// storage error.
	//
	// For operations where the audit record is part of the contract, not a
	// side effect: an authorization change that is not evidenced is a change
	// nobody can account for. Everything else — logins, token issuance —
	// keeps LogEvent, so the audit table stays off those hot paths.
	//
	// The caller decides what a failure means. For authorization changes the
	// convention is to return the error and NOT compensate: by the time this
	// is called the change has already been applied, so the error means
	// "applied but unevidenced", not "nothing happened".
	LogEventSync(ctx context.Context, event Event) error
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

// buildAuditLog converts an Event into its storage row. Shared by LogEvent and
// LogEventSync so the two can never disagree about how a record is shaped —
// in particular, both fold Protocol into Metadata via metadataWithProtocol.
func buildAuditLog(event Event) *schemas.AuditLog {
	return &schemas.AuditLog{
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
}

// LogEvent asynchronously records an audit log entry.
func (p *provider) LogEvent(event Event) {
	asyncutil.Go(p.deps.Log, func() {
		log := p.deps.Log.With().Str("func", "LogEvent").Logger()
		if err := p.deps.StorageProvider.AddAuditLog(context.Background(), buildAuditLog(event)); err != nil {
			log.Debug().Err(err).Str("action", event.Action).Msg("Failed to add audit log")
		}
	})
}

// LogEventSync records an audit log entry synchronously.
func (p *provider) LogEventSync(ctx context.Context, event Event) error {
	return p.deps.StorageProvider.AddAuditLog(ctx, buildAuditLog(event))
}

package scim

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/audit"
)

type recordingAudit struct {
	audit.Provider
	got audit.Event
	err error
}

func (r *recordingAudit) LogEventSync(_ context.Context, e audit.Event) error {
	if r.err != nil {
		return r.err
	}
	r.got = e
	return nil
}

// scim's concrete type embeds Dependencies BY VALUE (`type provider struct {
// Dependencies }`), so fields are reached as p.AuditProvider, not p.deps.X.
func newAuditTestProvider(ap audit.Provider) *provider {
	log := zerolog.Nop()
	return &provider{Dependencies: Dependencies{Log: &log, AuditProvider: ap}}
}

// A deployment that does not wire audit must still serve SCIM, matching the
// documented EventsProvider convention.
func TestLogAuditSync_NilProviderIsNoOp(t *testing.T) {
	p := newAuditTestProvider(nil)
	assert.NotPanics(t, func() {
		require.NoError(t, p.logAuditSync(context.Background(), audit.Event{Action: "x"}))
	})
}

func TestLogAuditSync_ForwardsEvent(t *testing.T) {
	ra := &recordingAudit{}
	p := newAuditTestProvider(ra)

	require.NoError(t, p.logAuditSync(context.Background(), audit.Event{Action: "scim.group_members_added"}))

	assert.Equal(t, "scim.group_members_added", ra.got.Action)
}

func TestLogAuditSync_PropagatesError(t *testing.T) {
	boom := errors.New("audit unavailable")
	p := newAuditTestProvider(&recordingAudit{err: boom})

	assert.ErrorIs(t, p.logAuditSync(context.Background(), audit.Event{Action: "x"}), boom)
}

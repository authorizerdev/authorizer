package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// fakeAuditStore records what AddAuditLog received and can be told to fail.
// It embeds storage.Provider (nil) so it satisfies the interface without
// implementing the other ~200 methods; only AddAuditLog is ever called here.
type fakeAuditStore struct {
	storage.Provider
	got *schemas.AuditLog
	err error
}

func (f *fakeAuditStore) AddAuditLog(_ context.Context, l *schemas.AuditLog) error {
	if f.err != nil {
		return f.err
	}
	f.got = l
	return nil
}

func newTestProvider(st storage.Provider) Provider {
	log := zerolog.Nop()
	return New(&Dependencies{Log: &log, StorageProvider: st})
}

func TestLogEventSync_PersistsEventAndReturnsNil(t *testing.T) {
	st := &fakeAuditStore{}
	p := newTestProvider(st)

	err := p.LogEventSync(context.Background(), Event{
		Action:    "admin.fga_tuples_written",
		ActorType: "admin",
		ActorID:   "actor-1",
	})

	require.NoError(t, err)
	require.NotNil(t, st.got)
	assert.Equal(t, "admin.fga_tuples_written", st.got.Action)
	assert.Equal(t, "actor-1", st.got.ActorID)
}

func TestLogEventSync_ReturnsStorageError(t *testing.T) {
	boom := errors.New("audit table unavailable")
	p := newTestProvider(&fakeAuditStore{err: boom})

	err := p.LogEventSync(context.Background(), Event{Action: "admin.fga_reset"})

	require.Error(t, err)
	assert.ErrorIs(t, err, boom, "the storage error must reach the caller unchanged")
}

func TestLogEventSync_FoldsProtocolIntoMetadataLikeLogEvent(t *testing.T) {
	st := &fakeAuditStore{}
	p := newTestProvider(st)

	require.NoError(t, p.LogEventSync(context.Background(), Event{
		Action:   "admin.fga_tuples_written",
		Protocol: "grpc",
		Metadata: `{"count":2}`,
	}))

	require.NotNil(t, st.got)
	// Both keys must survive: the protocol the caller set, and the metadata it
	// already had. A separate builder for the sync path would drop one.
	assert.JSONEq(t, `{"protocol":"grpc","count":2}`, st.got.Metadata)
}

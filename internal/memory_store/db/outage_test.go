package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/memory_store/memerr"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// faultyStorage makes every getter the DB-backed store uses fail with err.
// The embedded nil Provider satisfies the rest of the interface.
type faultyStorage struct {
	storage.Provider
	err   error
	state *schemas.OAuthState
}

func (s *faultyStorage) CleanExpiredSessionTokens(context.Context) error { return nil }
func (s *faultyStorage) CleanExpiredMFASessions(context.Context) error   { return nil }
func (s *faultyStorage) GetSessionTokenByUserIDAndKey(context.Context, string, string) (*schemas.SessionToken, error) {
	return nil, s.err
}
func (s *faultyStorage) GetAllMFASessionsByUserID(context.Context, string) ([]*schemas.MFASession, error) {
	return nil, s.err
}
func (s *faultyStorage) GetAllMFASessions(context.Context) ([]*schemas.MFASession, error) {
	return nil, s.err
}
func (s *faultyStorage) GetOAuthStateByKey(context.Context, string) (*schemas.OAuthState, error) {
	if s.state != nil {
		return s.state, nil
	}
	return nil, s.err
}
func (s *faultyStorage) DeleteOAuthStateByKey(context.Context, string) (bool, error) {
	return s.state != nil, nil
}

func newFaultyDBStore(t *testing.T, st *faultyStorage) *provider {
	t.Helper()
	log := zerolog.Nop()
	p, err := NewDBProvider(&config.Config{}, &Dependencies{Log: &log, StorageProvider: st})
	require.NoError(t, err)
	return p
}

// TestDBStore_OutageIsNotAMiss: a storage fault must stay a fault. Mapping it
// to ErrNotFound would turn a database outage into "token not found" for the
// caller and drop it from error alerting (misses are logged at warn).
func TestDBStore_OutageIsNotAMiss(t *testing.T) {
	outage := &storage.Error{Err: errors.New("dial tcp 10.0.0.9:5432: connect: connection refused")}
	p := newFaultyDBStore(t, &faultyStorage{err: outage})

	_, err := p.GetUserSession("u", "k")
	assert.ErrorIs(t, err, outage)
	assert.NotErrorIs(t, err, memerr.ErrNotFound)

	_, err = p.GetMfaSession("u", "k")
	assert.NotErrorIs(t, err, memerr.ErrNotFound)
	_, err = p.GetAllMfaSessions("u")
	assert.NotErrorIs(t, err, memerr.ErrNotFound)
	_, _, err = p.GetMfaSessionOwner("k")
	assert.NotErrorIs(t, err, memerr.ErrNotFound)
	_, err = p.GetState("k")
	assert.NotErrorIs(t, err, memerr.ErrNotFound)
	_, err = p.GetAndRemoveState("k")
	assert.NotErrorIs(t, err, memerr.ErrNotFound)
}

// TestDBStore_StorageNotFoundIsAMiss: the storage layer's own not-found is
// still reported as a memory-store miss.
func TestDBStore_StorageNotFoundIsAMiss(t *testing.T) {
	p := newFaultyDBStore(t, &faultyStorage{err: &storage.Error{Err: gorm.ErrRecordNotFound}})
	_, err := p.GetUserSession("u", "k")
	assert.ErrorIs(t, err, memerr.ErrNotFound)
	_, err = p.GetState("k")
	assert.ErrorIs(t, err, memerr.ErrNotFound)
}

// TestDBStore_ExpiredStateIsAMiss: expiry is reported as a miss, as the
// in-memory and Redis stores do.
func TestDBStore_ExpiredStateIsAMiss(t *testing.T) {
	old := &schemas.OAuthState{Key: "k", State: "v", CreatedAt: time.Now().Add(-11 * time.Minute).Unix()}
	p := newFaultyDBStore(t, &faultyStorage{state: old})
	_, err := p.GetState("k")
	assert.ErrorIs(t, err, memerr.ErrNotFound)
	_, err = p.GetAndRemoveState("k")
	assert.ErrorIs(t, err, memerr.ErrNotFound)
}

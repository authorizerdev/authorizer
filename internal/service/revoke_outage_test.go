package service

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
)

// sessionStorage backs the DB memory store; every session lookup fails with err.
type sessionStorage struct {
	storage.Provider
	err error
}

func (s *sessionStorage) CleanExpiredSessionTokens(context.Context) error { return nil }
func (s *sessionStorage) GetSessionTokenByUserIDAndKey(context.Context, string, string) (*schemas.SessionToken, error) {
	return nil, s.err
}

type revokeToken struct{ token.Provider }

func (revokeToken) ParseJWTToken(string) (jwt.MapClaims, error) {
	return jwt.MapClaims{"sub": "user-1", "nonce": "n-1", "login_method": "basic_auth"}, nil
}

func newRevokeProvider(t *testing.T, storeErr error) *provider {
	t.Helper()
	log := zerolog.Nop()
	ms, err := memory_store.New(&config.Config{DatabaseType: "sqlite"}, &memory_store.Dependencies{
		Log:             &log,
		StorageProvider: &sessionStorage{err: storeErr},
	})
	require.NoError(t, err)
	return &provider{
		Config: &config.Config{},
		Dependencies: Dependencies{
			Log:                 &log,
			TokenProvider:       revokeToken{},
			MemoryStoreProvider: ms,
		},
	}
}

// TestRevoke_StorageOutageIsInternal: when the DB-backed session store cannot
// be read, revoke must report a masked internal error (500, logged at error),
// not "token not found" — an outage must not look like the caller's mistake.
func TestRevoke_StorageOutageIsInternal(t *testing.T) {
	p := newRevokeProvider(t, &storage.Error{Err: errors.New("dial tcp 10.0.0.9:5432: connect: connection refused")})
	_, _, err := p.Revoke(context.Background(), RequestMetadata{}, &model.OAuthRevokeRequest{RefreshToken: "rt"})
	require.Error(t, err)
	var se *Error
	assert.False(t, errors.As(err, &se), "an outage must not be a typed (NotFound) error")
	_, _, masked := MaskInternalError(err)
	assert.True(t, masked)
}

// TestRevoke_SessionMissIsNotFound: the storage layer's own not-found is a
// miss and keeps the typed NotFound answer.
func TestRevoke_SessionMissIsNotFound(t *testing.T) {
	p := newRevokeProvider(t, &storage.Error{Err: gorm.ErrRecordNotFound})
	_, _, err := p.Revoke(context.Background(), RequestMetadata{}, &model.OAuthRevokeRequest{RefreshToken: "rt"})
	requireKind(t, err, KindNotFound)
}

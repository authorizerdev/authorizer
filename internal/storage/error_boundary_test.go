package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// fakeBackend implements only the methods these tests call; the embedded nil
// Provider satisfies the rest of the interface.
type fakeBackend struct {
	Provider
	addErr error
}

func (f *fakeBackend) AddUser(_ context.Context, _ *schemas.User) (*schemas.User, error) {
	return nil, f.addErr
}

func (f *fakeBackend) GetUserByID(_ context.Context, _ string) (*schemas.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeBackend) GetClientByClientID(_ context.Context, _ string) (*schemas.Client, error) {
	return nil, nil
}

// TestErrorBoundary_MarksErrorsTransparently: every backend error comes out
// marked as a storage error, with its text and identity untouched so existing
// callers (IsNotFound, errors.Is on driver sentinels, message logs) keep working.
func TestErrorBoundary_MarksErrorsTransparently(t *testing.T) {
	ctx := context.Background()
	driverErr := errors.New("mssql: Cannot insert duplicate key row in object 'dbo.authorizer_users' with unique index 'idx_email'")
	p := &errorBoundary{next: &fakeBackend{addErr: driverErr}}

	_, err := p.AddUser(ctx, &schemas.User{})
	require.Error(t, err)
	assert.True(t, IsStorageError(err))
	assert.ErrorIs(t, err, driverErr)
	assert.Equal(t, driverErr.Error(), err.Error(), "message must be unchanged for server-side logs")

	_, err = p.GetUserByID(ctx, "x")
	require.Error(t, err)
	assert.True(t, IsStorageError(err))
	assert.True(t, IsNotFound(err), "not-found contract must survive the wrapper")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// GetClientByClientID's documented (nil, nil) must stay (nil, nil).
	c, err := p.GetClientByClientID(ctx, "x")
	assert.Nil(t, c)
	assert.NoError(t, err)
}

func TestIsStorageError_PlainErrorIsNot(t *testing.T) {
	assert.False(t, IsStorageError(nil))
	assert.False(t, IsStorageError(errors.New("invalid email")))
}

// TestNew_WrapsBackendErrors guards the wiring: the Provider returned by New
// must be the boundary, so a real backend error is marked.
func TestNew_WrapsBackendErrors(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	p, err := New(getTestDBConfig(constants.DbTypeSqlite), &Dependencies{Log: &logger})
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	_, err = p.GetUserByID(context.Background(), "absent-"+uuid.NewString())
	require.Error(t, err)
	assert.True(t, IsStorageError(err))
	assert.True(t, IsNotFound(err))
}

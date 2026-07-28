package sql

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestSQLContextCancellationIsHonored proves the WithContext(ctx) threading is
// real, not cosmetic: an already-cancelled context passed into a read must abort
// with a context error instead of running the query and returning a row. Before
// the fix every GORM call used the bare p.db handle, so the caller's context was
// silently dropped and this read would have succeeded despite the cancellation.
func TestSQLContextCancellationIsHonored(t *testing.T) {
	for _, dbType := range sqlMigrationTestDBTypes() {
		t.Run(dbType, func(t *testing.T) {
			cfg := sqlMigrationTestConfig(t, dbType)
			p, err := NewProvider(cfg, sqlTestDeps(t))
			require.NoError(t, err)
			defer func() { _ = p.Close() }()

			email := uuid.New().String() + "@example.com"
			_, err = p.AddUser(context.Background(), &schemas.User{
				ID:    uuid.New().String(),
				Email: refs.NewStringRef(email),
			})
			require.NoError(t, err)

			// Sanity: with a live context the user is found.
			found, err := p.GetUserByEmail(context.Background(), email)
			require.NoError(t, err)
			require.NotNil(t, found)

			// With an already-cancelled context the same read must fail fast with a
			// context error rather than ignoring cancellation and returning the row.
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err = p.GetUserByEmail(ctx, email)
			require.Error(t, err, "cancelled context must abort the query")
			assert.True(t, errors.Is(err, context.Canceled),
				"expected context.Canceled to propagate to the DB layer, got: %v", err)
		})
	}
}

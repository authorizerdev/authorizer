package sql

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestAddAuthenticatorUniqueUserMethod verifies fix #4: a genuine unique index
// on (user_id, method) backs AddAuthenticator, so an enrollment that slips past
// the check-then-insert pre-check upserts instead of creating a duplicate row.
func TestAddAuthenticatorUniqueUserMethod(t *testing.T) {
	for _, dbType := range sqlMigrationTestDBTypes() {
		t.Run(dbType, func(t *testing.T) {
			cfg := sqlMigrationTestConfig(t, dbType)
			p, err := NewProvider(cfg, sqlTestDeps(t))
			require.NoError(t, err)
			defer func() { _ = p.Close() }()

			ctx := context.Background()
			userID := uuid.New().String()
			method := constants.EnvKeyTOTPAuthenticator

			// First enrollment inserts a row.
			_, err = p.AddAuthenticator(ctx, &schemas.Authenticator{
				UserID: userID,
				Method: method,
				Secret: "secret-1",
			})
			require.NoError(t, err)

			// Simulate the race: a second concurrent enrollment that passed the
			// pre-check. This mirrors AddAuthenticator's OnConflict Create with a
			// fresh id and must NOT create a duplicate.
			second := &schemas.Authenticator{
				ID:        uuid.New().String(),
				Key:       uuid.New().String(),
				UserID:    userID,
				Method:    method,
				Secret:    "secret-2",
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			}
			res := p.db.Clauses(clause.OnConflict{
				UpdateAll: true,
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "method"}},
			}).Create(second)
			require.NoError(t, res.Error)

			// Exactly one row for (user_id, method).
			var count int64
			require.NoError(t, p.db.Model(&schemas.Authenticator{}).
				Where("user_id = ? AND method = ?", userID, method).
				Count(&count).Error)
			assert.Equal(t, int64(1), count, "must be a single authenticator row per (user_id, method)")

			// The unique index is genuinely enforced: a plain duplicate insert fails.
			err = p.db.Create(&schemas.Authenticator{
				ID:        uuid.New().String(),
				Key:       uuid.New().String(),
				UserID:    userID,
				Method:    method,
				Secret:    "secret-3",
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			}).Error
			require.Error(t, err, "unique index on (user_id, method) should reject a duplicate insert")
		})
	}
}

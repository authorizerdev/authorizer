package integration_tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestLoginAttempts tests AddLoginAttempt, CountFailedLoginAttemptsSince, and DeleteLoginAttemptsBefore
func TestLoginAttempts(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	ctx := context.Background()

	email := "ratelimit-test@example.com"
	ipAddress := "192.0.2.1"
	now := time.Now().Unix()
	oneHourAgo := now - 3600

	t.Run("should add a failed login attempt", func(t *testing.T) {
		attempt := &schemas.LoginAttempt{
			Email:       email,
			IPAddress:   ipAddress,
			Successful:  false,
			AttemptedAt: now,
		}
		err := ts.StorageProvider.AddLoginAttempt(ctx, attempt)
		require.NoError(t, err)
		assert.NotEmpty(t, attempt.ID)
		assert.Equal(t, attempt.Key, attempt.ID)
		assert.NotZero(t, attempt.CreatedAt)
	})

	t.Run("should add a successful login attempt", func(t *testing.T) {
		attempt := &schemas.LoginAttempt{
			Email:       email,
			IPAddress:   ipAddress,
			Successful:  true,
			AttemptedAt: now,
		}
		err := ts.StorageProvider.AddLoginAttempt(ctx, attempt)
		require.NoError(t, err)
		assert.NotEmpty(t, attempt.ID)
	})

	t.Run("should count only failed attempts since timestamp", func(t *testing.T) {
		count, err := ts.StorageProvider.CountFailedLoginAttemptsSince(ctx, email, oneHourAgo)
		require.NoError(t, err)
		// We added 1 failed attempt above
		assert.GreaterOrEqual(t, count, int64(1))
	})

	t.Run("should not count successful attempts", func(t *testing.T) {
		// Add a second email with only successful attempts
		successEmail := "success-only@example.com"
		attempt := &schemas.LoginAttempt{
			Email:       successEmail,
			IPAddress:   ipAddress,
			Successful:  true,
			AttemptedAt: now,
		}
		err := ts.StorageProvider.AddLoginAttempt(ctx, attempt)
		require.NoError(t, err)

		count, err := ts.StorageProvider.CountFailedLoginAttemptsSince(ctx, successEmail, oneHourAgo)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("should not count attempts before the since timestamp", func(t *testing.T) {
		oldEmail := "old-attempts@example.com"
		twoHoursAgo := now - 7200
		attempt := &schemas.LoginAttempt{
			Email:       oldEmail,
			IPAddress:   ipAddress,
			Successful:  false,
			AttemptedAt: twoHoursAgo,
		}
		err := ts.StorageProvider.AddLoginAttempt(ctx, attempt)
		require.NoError(t, err)

		// Count since one hour ago — the attempt is two hours old, should not be counted
		count, err := ts.StorageProvider.CountFailedLoginAttemptsSince(ctx, oldEmail, oneHourAgo)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("should delete attempts before timestamp", func(t *testing.T) {
		cleanupEmail := "cleanup@example.com"
		twoHoursAgo := now - 7200

		// Add an old failed attempt
		oldAttempt := &schemas.LoginAttempt{
			Email:       cleanupEmail,
			IPAddress:   ipAddress,
			Successful:  false,
			AttemptedAt: twoHoursAgo,
		}
		err := ts.StorageProvider.AddLoginAttempt(ctx, oldAttempt)
		require.NoError(t, err)

		// Add a recent failed attempt
		recentAttempt := &schemas.LoginAttempt{
			Email:       cleanupEmail,
			IPAddress:   ipAddress,
			Successful:  false,
			AttemptedAt: now,
		}
		err = ts.StorageProvider.AddLoginAttempt(ctx, recentAttempt)
		require.NoError(t, err)

		// Verify both attempts count before cleanup
		countBefore, err := ts.StorageProvider.CountFailedLoginAttemptsSince(ctx, cleanupEmail, twoHoursAgo-1)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, countBefore, int64(2))

		// Delete attempts older than one hour ago
		err = ts.StorageProvider.DeleteLoginAttemptsBefore(ctx, oneHourAgo)
		require.NoError(t, err)

		// Only the recent attempt should remain (not older than oneHourAgo)
		countAfter, err := ts.StorageProvider.CountFailedLoginAttemptsSince(ctx, cleanupEmail, twoHoursAgo-1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), countAfter)
	})
}

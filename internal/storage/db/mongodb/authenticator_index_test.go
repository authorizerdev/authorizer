package mongodb

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestAuthenticatorUniqueIndex verifies the unique (user_id, method) compound
// index rejects a duplicate MFA enrollment at the storage layer. It is the
// backstop for AddAuthenticator's check-then-insert race: two concurrent
// requests can both pass the pre-check and both insert. The pre-check cannot be
// reached through the Provider interface sequentially, so this test writes raw
// documents that bypass it and asserts the index surfaces a clean duplicate-key
// error instead of leaving two divergent enrollments.
//
// Runs only against a live MongoDB (TEST_DBS includes mongodb); skipped otherwise.
func TestAuthenticatorUniqueIndex(t *testing.T) {
	if !mongoSelected() {
		t.Skip("skipping: TEST_DBS does not include mongodb")
	}

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	cfg := &config.Config{
		DatabaseType: constants.DbTypeMongoDB,
		DatabaseURL:  "mongodb://localhost:27017",
		DatabaseName: "authorizer_idx_test_" + strings.ReplaceAll(uuid.New().String(), "-", ""),
	}
	p, err := NewProvider(cfg, &Dependencies{Log: &logger})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = p.db.Drop(ctx)
		_ = p.Close()
	})

	ctx := context.Background()
	coll := p.db.Collection(schemas.Collections.Authenticators, options.Collection())

	userID := uuid.New().String()
	now := time.Now().Unix()

	first := &schemas.Authenticator{
		ID:        uuid.New().String(),
		UserID:    userID,
		Method:    constants.EnvKeyTOTPAuthenticator,
		Secret:    "secret-1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	first.Key = first.ID
	_, err = coll.InsertOne(ctx, first)
	require.NoError(t, err, "first enrollment should insert")

	// Racy second insert: distinct _id, same (user_id, method) pair — the exact
	// document a concurrent request would create after passing the pre-check.
	second := &schemas.Authenticator{
		ID:        uuid.New().String(),
		UserID:    userID,
		Method:    constants.EnvKeyTOTPAuthenticator,
		Secret:    "secret-2",
		CreatedAt: now,
		UpdatedAt: now,
	}
	second.Key = second.ID
	_, err = coll.InsertOne(ctx, second)
	require.Error(t, err, "duplicate (user_id, method) enrollment must be rejected by the unique index")
	assert.True(t, mongo.IsDuplicateKeyError(err), "error must be a clean duplicate-key error, got: %v", err)

	// Exactly one enrollment persists — no divergent duplicate document.
	count, err := coll.CountDocuments(ctx, map[string]interface{}{"user_id": userID, "method": constants.EnvKeyTOTPAuthenticator})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "only the first enrollment must persist")
}

// mongoSelected reports whether the mongodb backend is part of the current
// TEST_DBS selection (empty means the default all-DB suite, which includes it).
func mongoSelected() bool {
	v := os.Getenv("TEST_DBS")
	if v == "" {
		return true
	}
	for _, p := range strings.Split(v, ",") {
		if strings.TrimSpace(p) == constants.DbTypeMongoDB {
			return true
		}
	}
	return false
}

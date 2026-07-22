package mongodb

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// mongoTestURL is where the storage test suite expects a live MongoDB.
const mongoTestURL = "mongodb://localhost:27017"

// hasIndexOn returns true if the collection has an index whose key includes the
// given field, optionally requiring it to be unique.
func hasIndexOn(t *testing.T, coll *mongo.Collection, field string, requireUnique bool) bool {
	t.Helper()
	ctx := context.Background()
	cursor, err := coll.Indexes().List(ctx)
	require.NoError(t, err)
	var idxs []bson.M
	require.NoError(t, cursor.All(ctx, &idxs))
	for _, idx := range idxs {
		key, ok := idx["key"].(bson.M)
		if !ok {
			continue
		}
		if _, ok := key[field]; !ok {
			continue
		}
		if requireUnique {
			if u, _ := idx["unique"].(bool); !u {
				continue
			}
		}
		return true
	}
	return false
}

// TestUserEmailAndPhoneIndexesCreated proves the fix for the silently-broken
// user index batch: before the fix the phone_number spec combined sparse +
// partialFilterExpression, which MongoDB rejects, failing the whole all-or-
// nothing CreateMany and taking the email unique index with it. This test would
// have failed on the old code (neither unique index would exist) and passes now
// that the two indexes are created in separate, valid calls.
//
// Runs only against a live MongoDB (TEST_DBS includes mongodb); skipped otherwise.
func TestUserEmailAndPhoneIndexesCreated(t *testing.T) {
	if !mongoSelected() {
		t.Skip("skipping: TEST_DBS does not include mongodb")
	}

	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	cfg := &config.Config{
		DatabaseType: constants.DbTypeMongoDB,
		DatabaseURL:  mongoTestURL,
		DatabaseName: "authorizer_uidx_test_" + strings.ReplaceAll(uuid.New().String(), "-", ""),
	}
	p, err := NewProvider(cfg, &Dependencies{Log: &logger})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = p.db.Drop(ctx)
		_ = p.Close()
	})

	coll := p.db.Collection(schemas.Collections.User, options.Collection())
	assert.True(t, hasIndexOn(t, coll, "email", true), "email unique index must exist after NewProvider")
	assert.True(t, hasIndexOn(t, coll, "phone_number", true), "phone_number unique index must exist after NewProvider")

	ctx := context.Background()

	// Two phone-only users leave email nil (stored as BSON null). A unique+sparse
	// index would collide on the second null; the partialFilterExpression must
	// let both coexist while still enforcing uniqueness across real emails.
	u1 := &schemas.User{ID: uuid.New().String(), PhoneNumber: strPtr("+100000001")}
	u1.Key = u1.ID
	u2 := &schemas.User{ID: uuid.New().String(), PhoneNumber: strPtr("+100000002")}
	u2.Key = u2.ID
	_, err = coll.InsertOne(ctx, u1)
	require.NoError(t, err, "first phone-only user should insert")
	_, err = coll.InsertOne(ctx, u2)
	require.NoError(t, err, "second phone-only user (email null) must NOT collide on the email index")

	// A duplicate real email must still be rejected by the unique index.
	e1 := &schemas.User{ID: uuid.New().String(), Email: strPtr("dup@example.com")}
	e1.Key = e1.ID
	e2 := &schemas.User{ID: uuid.New().String(), Email: strPtr("dup@example.com")}
	e2.Key = e2.ID
	_, err = coll.InsertOne(ctx, e1)
	require.NoError(t, err)
	_, err = coll.InsertOne(ctx, e2)
	require.Error(t, err, "duplicate email must be rejected by the unique index")
	assert.True(t, mongo.IsDuplicateKeyError(err), "expected duplicate-key error, got: %v", err)
}

// TestUserEmailIndexWithPreexistingDuplicatesDoesNotCrash simulates an upgraded
// self-hosted deployment whose authorizer_users collection already holds
// duplicate emails (possible only because the unique index was silently never
// created). NewProvider must NOT fail — it must start and log a clear,
// actionable warning naming the duplicate-email problem.
func TestUserEmailIndexWithPreexistingDuplicatesDoesNotCrash(t *testing.T) {
	if !mongoSelected() {
		t.Skip("skipping: TEST_DBS does not include mongodb")
	}

	dbName := "authorizer_dupidx_test_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// Seed duplicate emails BEFORE NewProvider runs, using a raw client so the
	// unique index does not yet exist.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoTestURL))
	require.NoError(t, err)
	require.NoError(t, client.Ping(ctx, readpref.Primary()))
	rawColl := client.Database(dbName).Collection(schemas.Collections.User)
	_, err = rawColl.InsertMany(ctx, []interface{}{
		&schemas.User{ID: uuid.New().String(), Email: strPtr("clash@example.com")},
		&schemas.User{ID: uuid.New().String(), Email: strPtr("clash@example.com")},
	})
	require.NoError(t, err)
	require.NoError(t, client.Disconnect(ctx))

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	cfg := &config.Config{
		DatabaseType: constants.DbTypeMongoDB,
		DatabaseURL:  mongoTestURL,
		DatabaseName: dbName,
	}
	p, err := NewProvider(cfg, &Dependencies{Log: &logger})
	require.NoError(t, err, "NewProvider must not crash when duplicate emails already exist")
	t.Cleanup(func() {
		dctx, dcancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dcancel()
		_ = p.db.Drop(dctx)
		_ = p.Close()
	})

	logs := buf.String()
	assert.Contains(t, logs, "failed to create unique index on authorizer_users(email)",
		"a clear warning naming the duplicate-email problem must be logged")

	// The unique email index must NOT exist (build failed on the duplicates), so
	// uniqueness is enforced only at the application layer for now.
	coll := p.db.Collection(schemas.Collections.User, options.Collection())
	assert.False(t, hasIndexOn(t, coll, "email", true),
		"unique email index must not exist while duplicates remain")
}

func strPtr(s string) *string { return &s }

package storage

import (
	"github.com/authorizerdev/authorizer/internal/storage/db/arangodb"
	"github.com/authorizerdev/authorizer/internal/storage/db/cassandradb"
	"github.com/authorizerdev/authorizer/internal/storage/db/couchbase"
	"github.com/authorizerdev/authorizer/internal/storage/db/dynamodb"
	"github.com/authorizerdev/authorizer/internal/storage/db/mongodb"
	"github.com/authorizerdev/authorizer/internal/storage/db/sql"
)

// IsNotFound reports whether err means "the row does not exist", as opposed to
// "the query failed". Use it to keep those two apart:
//
//	org, err := p.StorageProvider.GetOrganizationByID(ctx, id)
//	switch {
//	case storage.IsNotFound(err):
//	    return nil, NotFound("organization not found")   // 404, permanent
//	case err != nil:
//	    return nil, Internal("storage unavailable")      // 500, retryable
//	}
//
// Why this matters: without it every caller collapses to `if err != nil` and
// reports a database outage as though the caller's input were wrong. A user
// clicking a perfectly valid verification link during a brief outage was told
// "invalid verification token" — a permanent, non-retryable answer to a
// transient condition, and in auth paths that ambiguity is a security concern
// as much as a UX one.
//
// It is a predicate rather than a single shared sentinel for two reasons. Each
// backend reports absence with its own driver value (gorm.ErrRecordNotFound,
// mongo.ErrNoDocuments, gocql.ErrNotFound, gocb.ErrDocumentNotFound), and
// errors.Is cannot match those against a foreign sentinel. And a sentinel
// declared here could not be wrapped by the backends anyway — this package
// imports all of them, so the dependency only runs one way. The shape follows
// k8s.io/apimachinery's apierrors.IsNotFound for the same reasons.
//
// Backends that have no canonical driver sentinel (ArangoDB, DynamoDB) declare
// their own and wrap it; see their errors.go.
//
// A nil error is never "not found".
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return sql.IsNotFound(err) ||
		mongodb.IsNotFound(err) ||
		arangodb.IsNotFound(err) ||
		cassandradb.IsNotFound(err) ||
		dynamodb.IsNotFound(err) ||
		couchbase.IsNotFound(err)
}

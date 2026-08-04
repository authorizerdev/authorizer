package storage

import (
	"errors"
	"fmt"
	"testing"

	"github.com/couchbase/gocb/v2"
	"github.com/gocql/gocql"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/storage/db/arangodb"
	"github.com/authorizerdev/authorizer/internal/storage/db/dynamodb"
)

// TestIsNotFoundRecognisesEveryBackend pins that the predicate actually matches
// each backend's absence value. If a backend is missed, its callers silently
// fall through to the "storage unavailable" branch and report a 500 for a row
// that simply does not exist — the inverse of the bug this replaced, and just
// as invisible.
func TestIsNotFoundRecognisesEveryBackend(t *testing.T) {
	cases := []struct {
		backend string
		err     error
	}{
		{"sql/gorm", gorm.ErrRecordNotFound},
		{"mongodb", mongo.ErrNoDocuments},
		{"cassandradb", gocql.ErrNotFound},
		{"couchbase", gocb.ErrDocumentNotFound},
		{"arangodb", arangodb.ErrNotFound},
		{"dynamodb", dynamodb.ErrNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.backend, func(t *testing.T) {
			if !IsNotFound(tc.err) {
				t.Fatalf("IsNotFound(%v) = false, want true — %s absences would be reported as storage faults", tc.err, tc.backend)
			}
		})
	}
}

// TestIsNotFoundSurvivesWrapping pins that the predicate still works once a
// backend adds context with %w, which is how the Arango/Dynamo sites report
// which resource was missing.
func TestIsNotFoundSurvivesWrapping(t *testing.T) {
	wrapped := fmt.Errorf("authenticator not found: %w", arangodb.ErrNotFound)
	if !IsNotFound(wrapped) {
		t.Fatal("IsNotFound must see through %w wrapping")
	}
	doubly := fmt.Errorf("loading user: %w", fmt.Errorf("row: %w", gorm.ErrRecordNotFound))
	if !IsNotFound(doubly) {
		t.Fatal("IsNotFound must see through nested wrapping")
	}
}

// TestIsNotFoundRejectsOtherErrors pins the other half of the contract: a real
// storage fault must NOT be mistaken for an absent row, or an outage would be
// reported to the user as "no such record" and silently swallowed.
func TestIsNotFoundRejectsOtherErrors(t *testing.T) {
	for _, err := range []error{
		nil,
		errors.New("connection refused"),
		errors.New("context deadline exceeded"),
		fmt.Errorf("dial tcp: %w", errors.New("i/o timeout")),
		// Deliberately similar wording, but not a sentinel — string-matching
		// "not found" would wrongly pass this.
		errors.New("host not found"),
	} {
		if IsNotFound(err) {
			t.Fatalf("IsNotFound(%v) = true, want false", err)
		}
	}
}

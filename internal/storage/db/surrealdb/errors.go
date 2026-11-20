package surrealdb

import (
	"errors"
	"strings"

	surrealdb "github.com/surrealdb/surrealdb.go/v2"
)

// ErrNotFound is returned (wrapped) by single-entity getters when no record
// matches. SurrealDB reports a missing record as an empty result rather than
// an error, so the provider supplies its own sentinel, as the ArangoDB and
// DynamoDB providers do.
var ErrNotFound = errors.New("surrealdb: record not found")

// IsNotFound reports whether err is (or wraps) ErrNotFound.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// The messages below are how SurrealDB 4.0 reports these conditions: as a
// *QueryError carrying only a message, with no typed kind (the SDK's
// IsTransactionConflict does not recognise them). TestErrorMessagesArePinned
// fails if a server or SDK upgrade changes them.
const (
	duplicateMarker = "already contains"
	indexMarker     = "Database index"
	conflictMarker  = "This transaction can be retried"
	recordMarker    = "Database record"
	existsMarker    = "already exists"
)

func queryErrorMessage(err error) (string, bool) {
	var qe *surrealdb.QueryError
	if errors.As(err, &qe) {
		return qe.Message, true
	}
	return "", false
}

// isDuplicate reports whether err is a unique-index violation.
func isDuplicate(err error) bool {
	msg, ok := queryErrorMessage(err)
	return ok && strings.Contains(msg, indexMarker) && strings.Contains(msg, duplicateMarker)
}

// isRecordExists reports whether err is a CREATE of a record id that is
// already taken.
func isRecordExists(err error) bool {
	msg, ok := queryErrorMessage(err)
	return ok && strings.Contains(msg, recordMarker) && strings.Contains(msg, existsMarker)
}

// isAuthExpired reports whether the server refused the session's
// credentials: an expired token (HTTP answers 401) or an invalid session.
func isAuthExpired(err error) bool {
	if err == nil {
		return false
	}
	if surrealdb.IsTokenExpired(err) || surrealdb.IsInvalidAuth(err) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "401 Unauthorized") || strings.Contains(msg, "problem with authentication")
}

// isConflict reports whether err is a retryable transaction conflict.
func isConflict(err error) bool {
	if surrealdb.IsTransactionConflict(err) {
		return true
	}
	msg, ok := queryErrorMessage(err)
	return ok && strings.Contains(msg, conflictMarker)
}

package arangodb

import "errors"

// ErrNotFound is this backend's canonical "row does not exist" error. Every
// not-found return wraps it (with a resource-specific message) so callers can
// tell an absent row from a storage fault via storage.IsNotFound, instead of
// treating any error as "absent" — which reports a database outage to the user
// as a permanently invalid credential.
//
// Backends deliberately own their own sentinel rather than importing one from
// internal/storage: that package imports every backend, so a shared sentinel
// there would be an import cycle. storage.IsNotFound fans out to each backend's
// predicate instead.
var ErrNotFound = errors.New("arangodb: record not found")

// IsNotFound reports whether err means "no such row" in this backend.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

package dynamodb

import "errors"

// ErrNotFound is this backend's canonical "row does not exist" error. See
// arangodb/errors.go for why each backend owns its own sentinel rather than
// sharing one from internal/storage (import cycle).
var ErrNotFound = errors.New("dynamodb: record not found")

// IsNotFound reports whether err means "no such row" in this backend.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

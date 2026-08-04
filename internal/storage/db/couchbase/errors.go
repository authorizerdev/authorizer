package couchbase

import (
	"errors"

	"github.com/couchbase/gocb/v2"
)

// IsNotFound reports whether err means "no such row" in this backend. gocb's
// ErrDocumentNotFound is canonical, so it is matched directly.
func IsNotFound(err error) bool {
	return errors.Is(err, gocb.ErrDocumentNotFound)
}

package cassandradb

import (
	"errors"

	"github.com/gocql/gocql"
)

// IsNotFound reports whether err means "no such row" in this backend. gocql's
// ErrNotFound is canonical, so it is matched directly.
func IsNotFound(err error) bool {
	return errors.Is(err, gocql.ErrNotFound)
}

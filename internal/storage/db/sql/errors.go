package sql

import (
	"errors"

	"gorm.io/gorm"
)

// IsNotFound reports whether err means "no such row" in this backend. GORM
// already has a single canonical sentinel, so there is nothing to wrap — the
// bare driver error returned by First()/Take() is matched directly.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

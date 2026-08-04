package mongodb

import (
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
)

// IsNotFound reports whether err means "no such row" in this backend. The
// driver's ErrNoDocuments is canonical, so it is matched directly.
func IsNotFound(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments)
}

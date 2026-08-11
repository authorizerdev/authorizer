package couchbase

import (
	"errors"

	"github.com/couchbase/gocb/v2"
)

// IsNotFound reports whether err means "no such row" in this backend.
//
// Two sentinels, because this provider reads through two different gocb APIs.
// Key-value gets return ErrDocumentNotFound; N1QL queries that matched nothing
// surface as ErrNoResult from Result.One(). Matching only the first meant every
// query-based getter — GetUserByID and GetClientByID among them — reported
// absence as an unrecognised error, so callers using storage.IsNotFound to
// separate "no such row" from "the query failed" got the wrong answer on this
// backend alone.
func IsNotFound(err error) bool {
	return errors.Is(err, gocb.ErrDocumentNotFound) || errors.Is(err, gocb.ErrNoResult)
}

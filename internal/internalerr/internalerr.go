// Package internalerr marks errors that came out of an infrastructure
// dependency (memory store / Redis, the OpenFGA datastore, JWT signing keys,
// SMTP and SMS providers, outbound HTTP) as internal: their text is useful in
// server logs and must never be shown to an API client verbatim.
//
// It is the non-storage counterpart of storage.Error. Providers mark errors at
// their own boundary with Wrap; every transport then replaces a marked error
// with a generic "internal server error (error_id: …)" message and logs the
// original (see service.MaskInternalError). Errors that are not marked are left
// alone, so deliberate user-facing messages keep reaching clients unchanged.
//
// This package is a leaf (standard library only) so any provider can import it
// without an import cycle.
package internalerr

import "errors"

// Error marks Err as an internal fault originating in Source (e.g.
// "memory_store", "openfga", "token", "email", "sms").
//
// Error() and Unwrap() are transparent: the message is unchanged and
// errors.Is / errors.As see straight through the wrapper, so callers that
// branch on a driver sentinel (redis.Nil, context.DeadlineExceeded, ...) keep
// working.
type Error struct {
	Source string
	Err    error
}

// Error returns the underlying error's text unchanged.
func (e *Error) Error() string { return e.Err.Error() }

// Unwrap exposes the underlying error.
func (e *Error) Unwrap() error { return e.Err }

// Wrap marks err as internal to source. A nil err stays nil, and an error that
// is already marked is returned as-is so the innermost source is kept.
func Wrap(source string, err error) error {
	if err == nil {
		return nil
	}
	if Is(err) {
		return err
	}
	return &Error{Source: source, Err: err}
}

// Is reports whether err (or anything it wraps) was marked internal.
func Is(err error) bool {
	var ie *Error
	return errors.As(err, &ie)
}

// Source returns the source of the outermost marker in err's chain, or "" when
// err is not marked.
func Source(err error) string {
	var ie *Error
	if errors.As(err, &ie) {
		return ie.Source
	}
	return ""
}

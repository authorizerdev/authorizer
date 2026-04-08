package token

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestErrBackchannelURIEmpty verifies the exported sentinel can be
// matched with errors.Is, allowing callers to discriminate the empty-URI
// case from generic failures.
func TestErrBackchannelURIEmpty(t *testing.T) {
	assert.True(t, errors.Is(ErrBackchannelURIEmpty, ErrBackchannelURIEmpty))

	wrapped := &wrappedErr{err: ErrBackchannelURIEmpty}
	assert.True(t, errors.Is(wrapped, ErrBackchannelURIEmpty))
}

// wrappedErr is a tiny helper to verify the sentinel is unwrap-friendly.
type wrappedErr struct{ err error }

func (w *wrappedErr) Error() string { return "wrapped: " + w.err.Error() }
func (w *wrappedErr) Unwrap() error { return w.err }

// TestSanitizeHTTPError verifies the sanitize helper redacts the full
// URL so callers can safely log net/http errors without leaking the
// path or query of the back-channel logout URI.
func TestSanitizeHTTPError(t *testing.T) {
	original := errors.New("Post \"https://rp.example.com/bcl?token=secret\": dial tcp: lookup failed")
	sanitized := sanitizeHTTPError(original, "https://rp.example.com/bcl?token=secret")
	assert.NotContains(t, sanitized.Error(), "secret")
	assert.NotContains(t, sanitized.Error(), "rp.example.com/bcl")
	assert.Contains(t, sanitized.Error(), "<redacted>")
	assert.Nil(t, sanitizeHTTPError(nil, "any"))
}

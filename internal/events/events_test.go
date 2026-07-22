package events

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Webhook custom headers are stored as a JSON object (the GraphQL Map scalar),
// so an admin can persist a value that is a number, bool, null or nested object
// (e.g. {"X-Retry": 3}). RegisterEvent's header loop previously did
// val.(string) — a single-return assertion that PANICS on any non-string value.
// That loop runs inside a bare `go func()` (see the RegisterEvent call sites),
// so the panic is unrecovered and crashes the whole process.
//
// headerValueString coerces the value instead. These cases cover the exact
// inputs that used to panic.
func TestHeaderValueString_NonStringDoesNotPanic(t *testing.T) {
	require.NotPanics(t, func() {
		require.Equal(t, "hello", headerValueString("hello"))
		require.Equal(t, "", headerValueString(nil))
		require.Equal(t, "3", headerValueString(float64(3))) // JSON numbers decode to float64
		require.Equal(t, "true", headerValueString(true))
	})
}

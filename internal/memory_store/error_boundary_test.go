package memory_store

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/internalerr"
)

// TestNew_MarksErrorsInternal: every provider New returns must mark its
// errors internal so transports mask them (issue #356) — the Redis and
// in-memory backends previously leaked their text verbatim, while the
// DB-backed store was masked as a storage error.
func TestNew_MarksErrorsInternal(t *testing.T) {
	log := zerolog.Nop()
	p, err := New(&config.Config{}, &Dependencies{Log: &log})
	require.NoError(t, err)

	_, err = p.GetUserSession("user", "missing")
	require.Error(t, err)
	assert.True(t, internalerr.Is(err))
	assert.Equal(t, "memory_store", internalerr.Source(err))
	assert.Equal(t, "not found", err.Error(), "message unchanged for server-side logs")

	// Success paths are untouched.
	require.NoError(t, p.SetState("k", "v"))
	v, err := p.GetState("k")
	require.NoError(t, err)
	assert.Equal(t, "v", v)
}

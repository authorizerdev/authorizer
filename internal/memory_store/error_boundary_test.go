package memory_store

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/internalerr"
)

// TestNew_MarksErrorsInternal: every provider New returns must mark its
// errors internal so transports mask them (issue #356) — before, errors from
// all three backends reached API clients verbatim.
func TestNew_MarksErrorsInternal(t *testing.T) {
	for _, storeType := range memoryStoreTypesForTest() {
		t.Run(storeType, func(t *testing.T) {
			p, err := newTestMemoryStore(t, storeType)
			require.NoError(t, err)

			_, err = p.GetUserSession("user", "missing")
			require.Error(t, err)
			assert.True(t, internalerr.Is(err))
			assert.Equal(t, "memory_store", internalerr.Source(err))

			// Success paths are untouched.
			require.NoError(t, p.SetState("k", "v"))
			v, err := p.GetState("k")
			require.NoError(t, err)
			assert.Equal(t, "v", v)
		})
	}
}

// TestIsNotFound_UniformAcrossBackends: a miss is distinguishable from a
// fault on every backend (ErrNotFound for in-memory/DB, redis.Nil for Redis),
// so callers for which "absent" is an expected outcome can classify it.
func TestIsNotFound_UniformAcrossBackends(t *testing.T) {
	for _, storeType := range memoryStoreTypesForTest() {
		t.Run(storeType, func(t *testing.T) {
			p, err := newTestMemoryStore(t, storeType)
			require.NoError(t, err)

			_, err = p.GetUserSession("user", "missing")
			assert.True(t, IsNotFound(err), "session miss: %v", err)
			_, err = p.GetMfaSession("user", "missing")
			assert.True(t, IsNotFound(err), "mfa session miss: %v", err)

			// Expiry is a miss too, not a distinct error.
			require.NoError(t, p.SetUserSession("user", "short", "tok", time.Now().Unix()+1))
			require.NoError(t, p.SetMfaSession("user", "short", "login", time.Now().Unix()+1))
			time.Sleep(2100 * time.Millisecond)
			_, err = p.GetUserSession("user", "short")
			assert.True(t, IsNotFound(err), "expired session: %v", err)
			_, err = p.GetMfaSession("user", "short")
			assert.True(t, IsNotFound(err), "expired mfa session: %v", err)
		})
	}
	assert.False(t, IsNotFound(errors.New("dial tcp 10.0.0.5:6379: connect: connection refused")))
	assert.False(t, IsNotFound(nil))
}

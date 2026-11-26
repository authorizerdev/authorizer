package internalerr

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrap(t *testing.T) {
	assert.NoError(t, Wrap("x", nil))

	base := fmt.Errorf("redis get: %w", context.DeadlineExceeded)
	err := Wrap("memory_store", base)
	assert.True(t, Is(err))
	assert.Equal(t, base.Error(), err.Error(), "message is unchanged")
	assert.ErrorIs(t, err, context.DeadlineExceeded, "sentinels stay reachable")
	assert.Equal(t, "memory_store", Source(err))

	// Re-wrapping keeps the innermost source.
	assert.Equal(t, "memory_store", Source(Wrap("token", fmt.Errorf("sign: %w", err))))

	assert.False(t, Is(errors.New("invalid email")))
	assert.Empty(t, Source(errors.New("invalid email")))
}

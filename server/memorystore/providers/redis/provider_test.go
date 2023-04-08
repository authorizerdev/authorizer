package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/authorizerdev/authorizer/server/memorystore/providers"
)

func TestRedisProvider(t *testing.T) {
	p, err := NewRedisProvider("redis://127.0.0.1:6379")
	assert.NoError(t, err)
	providers.ProviderTests(t, p)
}

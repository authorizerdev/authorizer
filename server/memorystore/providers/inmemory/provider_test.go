package inmemory

import (
	"testing"

	"github.com/authorizerdev/authorizer/server/memorystore/providers"
	"github.com/stretchr/testify/assert"
)

func TestInMemoryProvider(t *testing.T) {
	p, err := NewInMemoryProvider()
	assert.NoError(t, err)
	providers.ProviderTests(t, p)
}

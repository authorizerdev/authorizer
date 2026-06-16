package authctx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrincipalContext(t *testing.T) {
	t.Run("missing principal", func(t *testing.T) {
		p, ok := FromContext(context.Background())
		assert.False(t, ok)
		assert.Nil(t, p)
	})

	t.Run("loads stored principal", func(t *testing.T) {
		want := &Principal{
			UserID:       "user-1",
			LoginMethod:  "basic_auth",
			Nonce:        "nonce-1",
			IsSuperAdmin: false,
		}
		ctx := WithPrincipal(context.Background(), want)
		got, ok := FromContext(ctx)
		require.True(t, ok)
		require.NotNil(t, got)
		assert.Equal(t, want, got)
	})
}

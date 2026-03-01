package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMeta tests the meta query returns correct flags based on config
func TestMeta(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	t.Run("should return meta with default config", func(t *testing.T) {
		meta, err := ts.GraphQLProvider.Meta(ctx)
		require.NoError(t, err)
		assert.NotNil(t, meta)
		assert.Equal(t, cfg.ClientID, meta.ClientID)
		assert.True(t, meta.IsSignUpEnabled)
		assert.True(t, meta.IsBasicAuthenticationEnabled)
	})

	t.Run("should reflect disabled signup", func(t *testing.T) {
		cfg2 := getTestConfig()
		cfg2.EnableSignup = false
		ts2 := initTestSetup(t, cfg2)
		_, ctx2 := createContext(ts2)

		meta, err := ts2.GraphQLProvider.Meta(ctx2)
		require.NoError(t, err)
		assert.NotNil(t, meta)
		assert.False(t, meta.IsSignUpEnabled)
	})

	t.Run("should reflect disabled basic auth", func(t *testing.T) {
		cfg2 := getTestConfig()
		cfg2.EnableBasicAuthentication = false
		ts2 := initTestSetup(t, cfg2)
		_, ctx2 := createContext(ts2)

		meta, err := ts2.GraphQLProvider.Meta(ctx2)
		require.NoError(t, err)
		assert.NotNil(t, meta)
		assert.False(t, meta.IsBasicAuthenticationEnabled)
	})
}

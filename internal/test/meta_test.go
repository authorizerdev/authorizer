package test

import (
	"context"
	"testing"

	"github.com/authorizerdev/authorizer/internal/resolvers"
	"github.com/stretchr/testify/assert"
)

func metaTests(t *testing.T, s TestSetup) {
	t.Helper()
	t.Run(`should get meta information`, func(t *testing.T) {
		ctx := context.Background()
		meta, err := resolvers.MetaResolver(ctx)
		assert.Nil(t, err)
		assert.False(t, meta.IsFacebookLoginEnabled)
		assert.False(t, meta.IsGoogleLoginEnabled)
		assert.False(t, meta.IsGithubLoginEnabled)
		assert.True(t, meta.IsEmailVerificationEnabled)
		assert.True(t, meta.IsBasicAuthenticationEnabled)
		assert.True(t, meta.IsMagicLinkLoginEnabled)
	})
}

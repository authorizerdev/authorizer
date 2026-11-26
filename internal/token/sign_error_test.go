package token

import (
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/internalerr"
)

// TestSignJWTToken_KeyErrorIsInternal: a signing failure is server
// misconfiguration and its text (key-parsing detail) must be masked by the
// transports, so it is marked internal (issue #356).
func TestSignJWTToken_KeyErrorIsInternal(t *testing.T) {
	p := &provider{config: &config.Config{JWTType: "RS256", JWTPrivateKey: "not a pem key"}}
	_, err := p.SignJWTToken(jwt.MapClaims{"sub": "u"})
	require.Error(t, err)
	assert.True(t, internalerr.Is(err))
	assert.Equal(t, "token", internalerr.Source(err))

	p = &provider{config: &config.Config{JWTType: "HS256", JWTSecret: "secret"}}
	signed, err := p.SignJWTToken(jwt.MapClaims{"sub": "u"})
	require.NoError(t, err)
	assert.NotEmpty(t, signed)
}

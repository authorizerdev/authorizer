package http_handlers

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestConfigRedirectURIsAreExactForTheReservedClient pins the flag that closes
// the OIDC Core §3.1.2.1 gap: the deployment's own client has a registry row but
// nothing ever writes RedirectURIs to it, so without --redirect-uris it always
// falls through to AllowedOrigins, which compares ORIGINS. Any path under an
// allowed host was therefore a valid redirect target.
//
// The unset case carries as much weight as the set one. Tightening the default
// would refuse the callback of every deployment that never listed its URIs, so
// "unset behaves exactly as before" is the compatibility promise this asserts.
func TestConfigRedirectURIsAreExactForTheReservedClient(t *testing.T) {
	const reserved = "reserved-client"

	newProvider := func(redirectURIs []string) *httpProvider {
		logger := zerolog.Nop()
		return &httpProvider{
			Config: &config.Config{
				ClientID:       reserved,
				AllowedOrigins: []string{"https://app.example.com"},
				RedirectURIs:   redirectURIs,
			},
			Dependencies: Dependencies{
				Log: &logger,
				StorageProvider: &redirectURIClientStore{client: &schemas.Client{
					ClientID: reserved,
					Kind:     constants.ClientKindInteractive,
				}},
			},
		}
	}

	t.Run("unset keeps the origin fallback", func(t *testing.T) {
		h := newProvider(nil)
		for _, uri := range []string{
			"https://app.example.com/callback",
			"https://app.example.com/any/other/path",
		} {
			got, err := h.checkClientRedirectURI(context.Background(), reserved, uri, "https://auth.example.com")
			require.NoError(t, err)
			assert.True(t, got.Valid, "origin fallback must still accept %s", uri)
		}
	})

	t.Run("set holds the reserved client to an exact match", func(t *testing.T) {
		h := newProvider([]string{"https://app.example.com/callback", " https://app.example.com/other "})

		exact := []string{"https://app.example.com/callback", "https://app.example.com/other"}
		for _, uri := range exact {
			got, err := h.checkClientRedirectURI(context.Background(), reserved, uri, "https://auth.example.com")
			require.NoError(t, err)
			assert.True(t, got.Valid, "registered URI must be accepted: %s", uri)
		}

		// The conformance failure this flag exists for: a path the operator
		// never listed, on an origin they did allow.
		refused := []string{
			"https://app.example.com/callback/extra",
			"https://app.example.com/unregistered",
			"https://app.example.com/callback?code=x",
		}
		for _, uri := range refused {
			got, err := h.checkClientRedirectURI(context.Background(), reserved, uri, "https://auth.example.com")
			require.NoError(t, err)
			assert.False(t, got.Valid, "unregistered URI must be refused: %s", uri)
		}
	})

	t.Run("only the reserved client is affected", func(t *testing.T) {
		// A different client_id has its own registry row with its own registered
		// URIs; the config list must not leak onto it in either direction.
		logger := zerolog.Nop()
		h := &httpProvider{
			Config: &config.Config{
				ClientID:       reserved,
				AllowedOrigins: []string{"https://app.example.com"},
				RedirectURIs:   []string{"https://app.example.com/callback"},
			},
			Dependencies: Dependencies{
				Log: &logger,
				StorageProvider: &redirectURIClientStore{client: &schemas.Client{
					ClientID:     "other-client",
					Kind:         constants.ClientKindInteractive,
					RedirectURIs: "https://other.example.com/cb",
				}},
			},
		}

		got, err := h.checkClientRedirectURI(context.Background(), "other-client", "https://other.example.com/cb", "https://auth.example.com")
		require.NoError(t, err)
		assert.True(t, got.Valid, "another client keeps its own registered URIs")

		got, err = h.checkClientRedirectURI(context.Background(), "other-client", "https://app.example.com/callback", "https://auth.example.com")
		require.NoError(t, err)
		assert.False(t, got.Valid, "the config list must not apply to another client")
	})
}

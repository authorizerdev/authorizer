package codestate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodeRoundTrip pins the field order. Every slot is positional, so a
// renumbering would silently move a redirect_uri into the resource check (or a
// client id into nothing at all) rather than fail to compile.
func TestCodeRoundTrip(t *testing.T) {
	t.Parallel()
	in := Code{
		Challenge:   "abc123::S256",
		Session:     "session-fingerprint",
		Nonce:       "nonce-value",
		RedirectURI: "https://app.example.com/cb?x=1&y=2",
		Resource:    "https://api.example.com/v1",
		ClientID:    "client-a",
	}
	got := DecodeCode(EncodeCode(in))
	assert.Equal(t, in, got)
}

// TestCodeDelimiterInValues is the reason every free-text field is escaped: a
// redirect_uri containing the delimiter would otherwise shift every field after
// it, so a crafted redirect could push an attacker-chosen client id into the
// slot the token endpoint compares against.
func TestCodeDelimiterInValues(t *testing.T) {
	t.Parallel()
	in := Code{
		Challenge:   "c",
		Session:     "s",
		Nonce:       "n",
		RedirectURI: "https://evil.example.com/cb?a=@@b",
		Resource:    "https://api.example.com/@@x",
		ClientID:    "client-a",
	}
	got := DecodeCode(EncodeCode(in))
	assert.Equal(t, in.RedirectURI, got.RedirectURI)
	assert.Equal(t, in.Resource, got.Resource)
	assert.Equal(t, "client-a", got.ClientID, "a delimiter in an earlier field must not shift the client id")
}

// TestDecodeCodeLegacyBlobs is the upgrade guarantee: authorization codes minted
// by an older build are still in the store when a new binary starts, and must
// stay redeemable. Missing trailing fields decode empty, and an empty ClientID
// means the token endpoint skips the binding check rather than rejecting.
func TestDecodeCodeLegacyBlobs(t *testing.T) {
	t.Parallel()

	t.Run("four segments (pre-resource, pre-client-id)", func(t *testing.T) {
		t.Parallel()
		got := DecodeCode("chal::S256@@sess@@nonce@@https%3A%2F%2Fapp.example.com%2Fcb")
		assert.Equal(t, "chal::S256", got.Challenge)
		assert.Equal(t, "sess", got.Session)
		assert.Equal(t, "nonce", got.Nonce)
		assert.Equal(t, "https://app.example.com/cb", got.RedirectURI)
		assert.Empty(t, got.Resource)
		assert.Empty(t, got.ClientID, "an old code carries no client binding, so the check must be skipped")
	})

	t.Run("five segments (pre-client-id)", func(t *testing.T) {
		t.Parallel()
		got := DecodeCode("@@sess@@nonce@@https%3A%2F%2Fapp.example.com%2Fcb@@https%3A%2F%2Fapi.example.com")
		assert.Empty(t, got.Challenge, "no PKCE is a valid state")
		assert.Equal(t, "https://api.example.com", got.Resource)
		assert.Empty(t, got.ClientID)
	})
}

func TestAuthorizeRoundTrip(t *testing.T) {
	t.Parallel()
	in := Authorize{
		Code:        "the-code",
		Challenge:   "chal::S256",
		Nonce:       "nonce",
		RedirectURI: "https://app.example.com/cb",
		Resource:    "https://api.example.com",
		ClientID:    "client-a",
	}
	assert.Equal(t, in, DecodeAuthorize(EncodeAuthorize(in)))
}

// TestDecodeAuthorizeBareNonce covers the no-code-flow form, which /authorize
// writes as a naked nonce with no delimiter at all.
func TestDecodeAuthorizeBareNonce(t *testing.T) {
	t.Parallel()
	require.False(t, HasCode("just-a-nonce"))
	got := DecodeAuthorize("just-a-nonce")
	assert.Equal(t, "just-a-nonce", got.Nonce)
	assert.Empty(t, got.Code)
	assert.Empty(t, got.ClientID)
}

func TestDecodeAuthorizeLegacyBlob(t *testing.T) {
	t.Parallel()
	// Five segments: an in-flight /authorize detour started before the upgrade.
	got := DecodeAuthorize("code@@chal@@nonce@@https%3A%2F%2Fapp.example.com%2Fcb@@")
	assert.True(t, HasCode("code@@chal@@nonce@@x@@"))
	assert.Equal(t, "code", got.Code)
	assert.Equal(t, "https://app.example.com/cb", got.RedirectURI)
	assert.Empty(t, got.ClientID)
}

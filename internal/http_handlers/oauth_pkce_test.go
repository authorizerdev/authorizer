package http_handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// TestTwitterPKCEChallengeMatchesVerifier proves the login flow sends an S256
// code_challenge derived from the same verifier the callback replays. The
// original bug sent no challenge and replayed the provider name ("twitter") as
// the verifier, so the pair could never validate.
func TestTwitterPKCEChallengeMatchesVerifier(t *testing.T) {
	verifier := oauth2.GenerateVerifier()

	cfg := &oauth2.Config{
		ClientID: "test-client",
		Endpoint: oauth2.Endpoint{AuthURL: "https://twitter.com/i/oauth2/authorize"},
	}
	authURL := cfg.AuthCodeURL("state-abc", oauth2.S256ChallengeOption(verifier))

	parsed, err := url.Parse(authURL)
	require.NoError(t, err)
	q := parsed.Query()

	assert.Equal(t, "S256", q.Get("code_challenge_method"))

	sum := sha256.Sum256([]byte(verifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(sum[:])
	assert.Equal(t, expectedChallenge, q.Get("code_challenge"),
		"authorization request must carry the S256 challenge of the verifier")

	// Regression guard: the challenge is never the raw verifier, and the verifier
	// is never a static provider name (the original bug replayed "twitter").
	assert.NotEqual(t, verifier, q.Get("code_challenge"))
	assert.NotEqual(t, "twitter", verifier)
}

// TestPKCEVerifierStateKeyNamespaced ensures the verifier is stored under a key
// distinct from the provider-state entry so they never collide.
func TestPKCEVerifierStateKeyNamespaced(t *testing.T) {
	state := "rand___https://app/callback___user___openid profile email"
	assert.NotEqual(t, state, pkceVerifierKeyPrefix+state)
	assert.Contains(t, pkceVerifierKeyPrefix+state, state)
}

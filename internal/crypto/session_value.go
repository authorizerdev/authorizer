package crypto

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

// sessionValueDigestPrefix marks a stored session value as a digest rather than
// the raw token. It is what makes the transition observable: a value without it
// was written by an older build and is still the token in the clear.
const sessionValueDigestPrefix = "sha256:"

// HashSessionValue converts a bearer value (access token, refresh token,
// session fingerprint hash) into what should actually be persisted in the
// session store.
//
// These were stored verbatim, so anyone who could read the store — a Redis
// dump, a replica, a backup, an SSRF into the cache — walked away with live,
// directly replayable access and refresh tokens. Storing a digest means a
// reader learns only that *some* token was issued, not one they can present.
// The store never needs the original: every consumer either checks a key exists
// or compares against a token the caller already supplied.
func HashSessionValue(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return sessionValueDigestPrefix + hex.EncodeToString(sum[:])
}

// VerifySessionValue reports whether presented matches what the store holds.
//
// Dual-read for the upgrade: a value written before the deploy is still the raw
// token, so it is compared directly, while anything carrying the digest prefix
// is compared as a digest. Without this every live session and refresh token
// would stop working the moment the new binary starts. The legacy branch can be
// deleted once no pre-upgrade session can still be within its TTL.
//
// Constant-time on both branches — the stored value is a credential either way.
func VerifySessionValue(presented, stored string) bool {
	if presented == "" || stored == "" {
		return false
	}
	if strings.HasPrefix(stored, sessionValueDigestPrefix) {
		return subtle.ConstantTimeCompare([]byte(HashSessionValue(presented)), []byte(stored)) == 1
	}
	// Legacy plaintext row.
	return subtle.ConstantTimeCompare([]byte(presented), []byte(stored)) == 1
}

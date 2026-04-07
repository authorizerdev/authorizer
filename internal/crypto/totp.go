package crypto

import (
	"strings"
)

// TOTPCipherPrefix marks AES-GCM ciphertext written by EncryptTOTPSecret.
// The prefix lets the read path transparently handle both legacy plaintext
// rows AND prefixed ciphertext, and lets a future migration recognise
// already-encrypted rows so it never re-encrypts them. The "v1" component
// reserves room for algorithm rotation later — bump to "enc:v2:" if the
// underlying cipher or KDF ever changes.
const TOTPCipherPrefix = "enc:v1:"

// EncryptTOTPSecret encrypts the TOTP shared secret with AES-256-GCM
// (using the existing EncryptAES helper, which derives the key via HKDF)
// and prepends TOTPCipherPrefix so it is recognisable as ciphertext.
//
// TOTP secrets are long-lived — they are enrolled once per user and used
// forever — so a reversible primitive is required (we need the original
// secret on every Validate call to compute the expected code).
func EncryptTOTPSecret(plain, key string) (string, error) {
	if plain == "" {
		return "", nil
	}
	ct, err := EncryptAES(key, plain)
	if err != nil {
		return "", err
	}
	return TOTPCipherPrefix + ct, nil
}

// DecryptTOTPSecret decrypts a value previously written by
// EncryptTOTPSecret. Legacy plaintext rows (no prefix) are returned
// unchanged so the read path keeps working during the rolling upgrade
// from a pre-encryption release.
func DecryptTOTPSecret(stored, key string) (string, error) {
	if !strings.HasPrefix(stored, TOTPCipherPrefix) {
		// Legacy plaintext row — return as-is. The caller may then
		// re-encrypt it via the lazy-migration path.
		return stored, nil
	}
	return DecryptAES(key, strings.TrimPrefix(stored, TOTPCipherPrefix))
}

// IsEncryptedTOTPSecret reports whether the stored value already carries
// the TOTPCipherPrefix marker. Used by lazy migration in the TOTP
// authenticator to decide whether to re-encrypt a row after a successful
// validation.
func IsEncryptedTOTPSecret(stored string) bool {
	return strings.HasPrefix(stored, TOTPCipherPrefix)
}

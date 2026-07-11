package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// TOTPCipherPrefix marks AES-GCM ciphertext written by EncryptTOTPSecret.
// The prefix lets the totp authenticator's read path tell new ciphertext
// rows apart from legacy plaintext rows so the lazy migration only acts
// on the latter. The "v1" component reserves room for algorithm rotation
// later — bump to "enc:v2:" if the underlying cipher or KDF ever changes.
const TOTPCipherPrefix = "enc:v1:"

// ErrTOTPSecretNotEncrypted is returned by DecryptTOTPSecret when the
// stored value does not carry the TOTPCipherPrefix marker. The totp
// authenticator catches this and falls back to the raw stored value as
// a legacy base32 secret (then migrates it on a successful Validate).
var ErrTOTPSecretNotEncrypted = errors.New("totp: stored secret is not in enc:v1: form")

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
// EncryptTOTPSecret. It is strict: the stored value MUST carry the
// TOTPCipherPrefix marker. Callers handling a row written by an older
// release should detect ErrTOTPSecretNotEncrypted and treat the raw
// stored value as a legacy base32 secret.
func DecryptTOTPSecret(stored, key string) (string, error) {
	if !strings.HasPrefix(stored, TOTPCipherPrefix) {
		return "", ErrTOTPSecretNotEncrypted
	}
	return DecryptAES(key, strings.TrimPrefix(stored, TOTPCipherPrefix))
}

// IsEncryptedTOTPSecret reports whether the stored value carries the
// TOTPCipherPrefix marker. Used by the totp authenticator's lazy
// migration to decide whether a row needs to be rewritten after a
// successful validation.
func IsEncryptedTOTPSecret(stored string) bool {
	return strings.HasPrefix(stored, TOTPCipherPrefix)
}

// HashRecoveryCode returns the hex-encoded SHA-256 digest of a TOTP
// recovery code. Recovery codes are generated as random UUIDs — already
// high-entropy — so a fast one-way hash is sufficient: there is nothing
// to brute-force offline the way there is with a user-chosen password, so
// the slow-hash cost of bcrypt/argon2 would be wasted. Storing only the
// digest means an offline DB dump no longer reveals usable recovery codes.
func HashRecoveryCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// IsHashedRecoveryCode reports whether a stored recovery-code map key is a
// SHA-256 digest (64 hex chars) written by HashRecoveryCode, as opposed to
// a legacy plaintext UUID (36 chars, contains dashes) written by a
// pre-hashing release. Used by the totp authenticator's lazy migration to
// decide whether an incoming code must be hashed before comparison.
func IsHashedRecoveryCode(stored string) bool {
	if len(stored) != hex.EncodedLen(sha256.Size) {
		return false
	}
	_, err := hex.DecodeString(stored)
	return err == nil
}

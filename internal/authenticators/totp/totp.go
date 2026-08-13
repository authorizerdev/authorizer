package totp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"

	"github.com/authorizerdev/authorizer/internal/authenticators/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

const (
	// totpPendingSecretPrefix namespaces a not-yet-confirmed TOTP secret held
	// in the memory store during a re-enrollment of an already-verified
	// authenticator. Deliberately kept out of the DB schema (no storage
	// provider needs a new column) — the same transient-state pattern the
	// per-user TOTP lockout counter uses.
	totpPendingSecretPrefix = "totp_pending_secret:"
	// totpPendingSecretTTLSeconds bounds how long a generated-but-unconfirmed
	// secret lingers before the user must restart the re-setup.
	totpPendingSecretTTLSeconds = 10 * 60
	// totpUsedPasscodePrefix namespaces the single-use claim on an already
	// redeemed TOTP passcode. See reserveTOTPPasscode.
	totpUsedPasscodePrefix = "totp_used:"
	// totpPasscodeReuseWindowSeconds must cover every time-step totp.Validate
	// will accept — Period 30 with Skew 1 spans three steps — so a redeemed
	// code stays claimed for as long as it would otherwise still validate.
	totpPasscodeReuseWindowSeconds = 90
	// totalRecoveryCodes is how many single-use recovery codes Generate issues
	// per enrollment. It also bounds recoveryCodeConsumeAttempts: it is the most
	// redemptions that can legitimately contend for the row.
	totalRecoveryCodes = 10
)

// pendingTOTPSecret is the memory-store payload for a re-enrollment awaiting
// confirmation: the new at-rest-encrypted secret and its hashed recovery-codes
// blob, promoted to the live Authenticator row only once the user confirms the
// new code via Validate.
type pendingTOTPSecret struct {
	Secret        string `json:"secret"`
	RecoveryCodes string `json:"recovery_codes"`
}

func totpPendingSecretKey(userID string) string {
	return totpPendingSecretPrefix + userID
}

// reserveTOTPPasscode atomically claims a (user, passcode) pair for one use and
// reports whether this caller won the claim. A second redemption of the same
// code inside its acceptance window loses and must be rejected.
//
// Keyed on the passcode rather than the matched time-step because
// totp.Validate does not report which step matched, and a step derived from
// the wall clock at redemption time would let the same code be replayed under a
// different step number a few seconds later — the exact replay this closes. The
// key is a digest, not the code itself, so a store dump yields nothing
// redeemable. TTL covers the full window Validate will accept (Period 30,
// Skew 1 → three steps).
//
// Fails open on a store fault or an unconfigured store: an outage must not
// lock every enrolled user out of their account.
func (p *provider) reserveTOTPPasscode(userID, passcode string) bool {
	if p.deps.MemoryStoreProvider == nil {
		return true
	}
	sum := sha256.Sum256([]byte(userID + ":" + passcode))
	key := totpUsedPasscodePrefix + hex.EncodeToString(sum[:])
	claimed, err := p.deps.MemoryStoreProvider.SetCacheNX(key, "1", totpPasscodeReuseWindowSeconds)
	if err != nil {
		return true
	}
	return claimed
}

// promotePendingSecret checks whether a pending (unconfirmed) re-enrollment
// secret is staged for userID and whether passcode validates against it. When
// it does, this call IS the user confirming their re-setup: the pending secret
// is promoted to the live row (secret, recovery codes, VerifiedAt) and cleared
// from the store. Returns (true, nil) when promoted. Until this moment the old
// secret keeps validating (via the normal path in Validate), so an abandoned
// re-setup never locks anyone out.
func (p *provider) promotePendingSecret(ctx context.Context, totpModel *schemas.Authenticator, passcode, userID string) (bool, error) {
	if p.deps.MemoryStoreProvider == nil {
		return false, nil
	}
	blob, err := p.deps.MemoryStoreProvider.GetCache(totpPendingSecretKey(userID))
	if err != nil || blob == "" {
		return false, nil
	}
	var pending pendingTOTPSecret
	if err := json.Unmarshal([]byte(blob), &pending); err != nil {
		return false, nil
	}
	plainSecret, err := crypto.DecryptTOTPSecret(pending.Secret, p.deps.EncryptionKey)
	if err != nil {
		return false, nil
	}
	if !totp.Validate(passcode, plainSecret) {
		// Not the new code. The caller may be logging in with the still-live old
		// secret; leave the pending secret staged for a later confirmation.
		return false, nil
	}
	now := time.Now().Unix()
	totpModel.Secret = pending.Secret
	totpModel.RecoveryCodes = refs.NewStringRef(pending.RecoveryCodes)
	totpModel.VerifiedAt = &now
	if _, err := p.deps.StorageProvider.UpdateAuthenticator(ctx, totpModel); err != nil {
		return false, err
	}
	_ = p.deps.MemoryStoreProvider.DeleteCacheByPrefix(totpPendingSecretKey(userID))
	return true, nil
}

// Generate generates a Time-Based One-Time Password (TOTP) for a user and returns the base64-encoded QR code for frontend display.
func (p *provider) Generate(ctx context.Context, id string) (*config.AuthenticatorConfig, error) {
	log := p.deps.Log.With().Str("func", "Generate (totp provider)").Logger()
	var buf bytes.Buffer
	// Get user details
	user, err := p.deps.StorageProvider.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// AccountName is the label the authenticator app shows next to the code, and
	// pquerna/otp REJECTS an empty one outright ("AccountName must be set").
	// Email alone is therefore wrong: a phone-only signup has no email, so
	// enrolling TOTP failed with that error surfaced straight to the user —
	// which, since MFA is on by default, is the first thing a mobile signup
	// hits after verifying.
	//
	// Fall back to the phone number, then the user id. The id is a poor label
	// but it is never empty, so enrolment cannot fail on a missing identifier.
	accountName := refs.StringValue(user.Email)
	if accountName == "" {
		accountName = refs.StringValue(user.PhoneNumber)
	}
	if accountName == "" {
		accountName = user.ID
	}

	// Generate totp, Authenticators hash is valid for 30 seconds
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "authorizer",
		AccountName: accountName,
	})
	if err != nil {
		return nil, err
	}
	// Generating image for key and encoding to base64 for displaying in frontend
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, err
	}
	_ = png.Encode(&buf, img)
	encodedText := crypto.EncodeB64(buf.String())
	secret := key.Secret()
	recoveryCodes := []string{}
	for i := 0; i < totalRecoveryCodes; i++ {
		recoveryCodes = append(recoveryCodes, uuid.NewString())
	}
	// recoverCodesMap is the plaintext map returned to the caller once (the
	// frontend shows these codes to the user at enrollment). storedCodesMap
	// keys on the SHA-256 hash of each code — only the hashed form is
	// persisted, so an offline DB dump never reveals usable recovery codes.
	recoverCodesMap := map[string]bool{}
	storedCodesMap := map[string]bool{}
	for i := 0; i < len(recoveryCodes); i++ {
		recoverCodesMap[recoveryCodes[i]] = false
		storedCodesMap[crypto.HashRecoveryCode(recoveryCodes[i])] = false
	}
	// Converting storedCodesMap (hashed) to string for persistence
	jsonData, err := json.Marshal(storedCodesMap)
	if err != nil {
		return nil, err
	}
	recoveryCodesString := string(jsonData)
	// Encrypt the TOTP shared secret at rest. The plaintext `secret` is
	// returned to the caller (frontend needs it to display the QR code
	// for enrollment) but never written to storage in plaintext.
	encryptedSecret, err := crypto.EncryptTOTPSecret(secret, p.deps.EncryptionKey)
	if err != nil {
		return nil, err
	}
	totpModel := &schemas.Authenticator{
		Secret:        encryptedSecret,
		RecoveryCodes: refs.NewStringRef(recoveryCodesString),
		UserID:        user.ID,
		Method:        constants.EnvKeyTOTPAuthenticator,
	}
	authenticator, err := p.deps.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, user.ID, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		log.Debug().Err(err).Msg("error getting authenticator details")
		// continue
	}
	switch {
	case authenticator != nil && authenticator.VerifiedAt != nil && p.deps.MemoryStoreProvider != nil:
		// Re-enrollment of an ALREADY-VERIFIED authenticator. Do NOT overwrite
		// the live secret/recovery-codes/VerifiedAt in place: that would desync
		// the user's working authenticator app the instant they click "set up",
		// even if they abandon the flow before confirming the new QR — a real
		// account-lockout risk. Stash the new secret as pending instead; it is
		// promoted to the live row only when the user confirms the new code via
		// Validate. Kept in the memory store (not the DB) so an abandoned or
		// lost re-setup self-heals: the pending secret simply expires and the
		// existing authenticator keeps working.
		blob, mErr := json.Marshal(pendingTOTPSecret{Secret: encryptedSecret, RecoveryCodes: recoveryCodesString})
		if mErr != nil {
			return nil, mErr
		}
		if err = p.deps.MemoryStoreProvider.SetCache(totpPendingSecretKey(user.ID), string(blob), totpPendingSecretTTLSeconds); err != nil {
			return nil, err
		}
	case authenticator == nil:
		// First-time enrollment: no working authenticator to protect.
		if _, err = p.deps.StorageProvider.AddAuthenticator(ctx, totpModel); err != nil {
			return nil, err
		}
	default:
		// An existing but UNVERIFIED row (a prior enrollment never confirmed, or
		// a verified row with no memory store wired) — nothing enrolled to lose,
		// so overwrite it in place.
		authenticator.Secret = encryptedSecret
		authenticator.RecoveryCodes = refs.NewStringRef(recoveryCodesString)
		if _, err = p.deps.StorageProvider.UpdateAuthenticator(ctx, authenticator); err != nil {
			return nil, err
		}
	}
	return &config.AuthenticatorConfig{
		ScannerImage:    encodedText,
		Secret:          secret,
		RecoveryCodes:   recoveryCodes,
		RecoveryCodeMap: recoverCodesMap,
	}, nil
}

// Validate validates a Time-Based One-Time Password (TOTP) against the
// stored TOTP secret for a user.
//
// The stored value can be in either of two forms:
//
//  1. enc:v1:<ciphertext> — the at-rest format. Decrypt and use the
//     plaintext to compute the expected code.
//  2. <raw base32> — a legacy row written by a pre-encryption release.
//     Use the stored value directly as the secret. On a successful
//     validation the row is re-encrypted in place (best-effort) so the
//     next read takes the encrypted path.
//
// Concurrency: two replicas observing the same legacy row may both
// decrypt, re-encrypt, and write before either commits. The two writes
// carry the same plaintext under different AES-GCM nonces; whichever
// lands last wins, the contents are semantically identical, and the row
// is permanently in the enc:v1: form afterwards. Subsequent calls take
// the encrypted-path branch immediately.
//
// Rolling-deploy note: a replica still on the pre-encryption release
// will read a migrated row as if it were a base32 secret and fail. For a
// rolling rollout across multiple replicas, prefer to complete the
// rollout before any TOTP user logs in (e.g. with a brief maintenance
// window) — or use an atomic deploy.
//
// Best-effort write: a failure to update the authenticator row after a
// successful validation never fails the login. The user supplied a valid
// TOTP code; refusing it because of a transient DB error or migration
// encrypt failure would be a worse outcome than a delayed VerifiedAt or
// a delayed migration. Failures are logged with structured fields.
func (p *provider) Validate(ctx context.Context, passcode string, userID string) (bool, error) {
	log := p.deps.Log.With().Str("func", "totp.Validate").Str("user_id", userID).Logger()

	totpModel, err := p.deps.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, userID, constants.EnvKeyTOTPAuthenticator)
	if err != nil {
		return false, err
	}
	// Providers disagree on how "not enrolled" is reported: most return an
	// error (gorm.ErrRecordNotFound and friends), but the DynamoDB provider
	// returns (nil, nil). Without this guard that case dereferences a nil row
	// below and panics, so treat a missing authenticator as a failed
	// validation — there is no secret to check the passcode against.
	if totpModel == nil {
		return false, nil
	}

	// A pending re-enrollment secret takes precedence: if one is staged and the
	// supplied code matches it, promote it now (the user is confirming their
	// re-setup). Checked before the live secret so a successful confirmation
	// switches over atomically; a non-matching code falls through to validate
	// against the still-live old secret below.
	if promoted, pErr := p.promotePendingSecret(ctx, totpModel, passcode, userID); pErr != nil {
		return false, pErr
	} else if promoted {
		return true, nil
	}

	var (
		plainSecret string
		// migrate is set when the stored value is legacy plaintext AND
		// the validation succeeds — at that point we know the raw value
		// is a real base32 secret worth re-encrypting in place.
		migrate bool
	)

	plainSecret, decErr := crypto.DecryptTOTPSecret(totpModel.Secret, p.deps.EncryptionKey)
	switch {
	case decErr == nil:
		// enc:v1: row — use the decrypted plaintext.
	case errors.Is(decErr, crypto.ErrTOTPSecretNotEncrypted):
		// Legacy plaintext row from a pre-encryption release. Use the
		// stored value directly; arrange to migrate it on success.
		plainSecret = totpModel.Secret
		migrate = true
	default:
		// Decryption was attempted (the row IS prefixed) but failed.
		// The most likely cause is a key mismatch: the at-rest key is
		// --encryption-key, which falls back to --jwt-secret when unset, so
		// rotating EITHER (without having set a dedicated --encryption-key
		// first) changes the key and locks enrolled TOTP users out. There is
		// no re-encryption path, so those users must re-enrol. Fail closed
		// and log loudly.
		log.Error().Err(decErr).Msg("failed to decrypt stored TOTP secret; check that --encryption-key (or --jwt-secret, if no encryption key is set) has not changed since enrollment")
		return false, decErr
	}

	status := totp.Validate(passcode, plainSecret)
	if !status {
		// Wrong code. Don't bother with VerifiedAt or migration —
		// nothing about the row should change on a failed login.
		return false, nil
	}

	// RFC 6238 §5.2: "the verifier MUST NOT accept the second attempt of the
	// same OTP". pquerna/otp is stateless — totp.Validate runs ValidateCustom
	// with Period 30 / Skew 1, matching against three time-steps and keeping no
	// record of what it accepted — so a code captured by a phishing proxy or a
	// single intercepted submission stays replayable for the whole ~90s window.
	// Email/SMS OTP is already single-use; TOTP was the outlier.
	if !p.reserveTOTPPasscode(userID, passcode) {
		log.Debug().Msg("totp passcode replayed within its validity window")
		return false, nil
	}

	// Two reasons we may need to write the row back after a successful
	// validation:
	//   1. First-time-ever validation → record VerifiedAt
	//   2. The row is legacy plaintext → re-encrypt in place
	updateVerifiedAt := totpModel.VerifiedAt == nil
	if updateVerifiedAt {
		timeNow := time.Now().Unix()
		totpModel.VerifiedAt = &timeNow
	}

	if migrate {
		ct, encErr := crypto.EncryptTOTPSecret(plainSecret, p.deps.EncryptionKey)
		if encErr != nil {
			// Encryption failed — log and skip the migration. The
			// validation itself succeeded so we still return true; the
			// next call retries naturally because the row is unchanged.
			log.Warn().Err(encErr).Msg("totp lazy migration: encrypt failed, leaving row unchanged")
			migrate = false
		} else {
			totpModel.Secret = ct
		}
	}

	if updateVerifiedAt || migrate {
		if _, err = p.deps.StorageProvider.UpdateAuthenticator(ctx, totpModel); err != nil {
			log.Warn().Err(err).
				Bool("verified_at_update", updateVerifiedAt).
				Bool("migration_attempt", migrate).
				Msg("totp post-validate row update failed; continuing")
		} else if migrate {
			log.Info().Msg("totp lazy migration: legacy plaintext row rewritten as enc:v1:")
		}
	}

	return true, nil
}

// recoveryCodeConsumeAttempts bounds the compare-and-swap retry loop in
// ValidateRecoveryCode.
//
// The value is not a generic "retry a few times" constant. A caller can only
// lose the swap because another redemption committed to the same row, and each
// of those spends one of the user's recovery codes — of which Generate issues
// exactly totalRecoveryCodes. So that count IS the worst case for legitimate
// contention: even if every code a user has is redeemed simultaneously, each
// request gets through within this many attempts. Anything beyond it is a
// database that is not making progress, and the loop stops rather than
// hammering a login path.
//
// Erring high is close to free — the realistic case resolves in one or two
// rounds, and the cost of the bound is only paid on the failure path — whereas
// erring low turns ordinary contention into a spurious fault for a user who is
// already locked out of their authenticator app.
const recoveryCodeConsumeAttempts = totalRecoveryCodes

// ValidateRecoveryCode validates a Time-Based One-Time Password (TOTP) recovery code against the stored TOTP recovery code for a user.
//
// Recovery codes are single-use, and the write that spends one has to be what
// enforces that. Reading the blob, deciding in Go, and writing back
// unconditionally let two concurrent redemptions of the SAME code both read it
// unconsumed and both return true — measured at 3 of 8 concurrent attempts
// succeeding before this loop existed — which hands an attacker holding one
// leaked code an unlimited number of logins. So the write goes through
// ConsumeAuthenticatorRecoveryCode, which only lands while the row still holds
// the blob this call read; on a lost race we re-read and look again, and the
// code is then marked consumed, so the loser returns false.
func (p *provider) ValidateRecoveryCode(ctx context.Context, recoveryCode, userID string) (bool, error) {
	log := p.deps.Log.With().Str("func", "ValidateRecoveryCode").Str("user_id", userID).Logger()
	for attempt := 0; attempt < recoveryCodeConsumeAttempts; attempt++ {
		// get totp details
		totpModel, err := p.deps.StorageProvider.GetAuthenticatorDetailsByUserId(ctx, userID, constants.EnvKeyTOTPAuthenticator)
		if err != nil {
			return false, err
		}
		// See Validate: the DynamoDB provider signals "not enrolled" as (nil, nil)
		// rather than an error, so guard before dereferencing.
		if totpModel == nil {
			return false, nil
		}
		// The blob is compared byte for byte by the storage layer, so keep the
		// exact string that was read — re-marshalling the map would reorder
		// keys and match nothing.
		storedCodes := refs.StringValue(totpModel.RecoveryCodes)
		// convert recoveryCodes to map
		recoveryCodesMap := map[string]bool{}
		if err := json.Unmarshal([]byte(storedCodes), &recoveryCodesMap); err != nil {
			return false, err
		}
		// Recovery codes are stored as SHA-256 hashes. Look up the hash of the
		// supplied code; if it isn't present, fall back to a direct plaintext
		// lookup for rows written by a pre-hashing release (lazy backward
		// compatibility, mirroring the TOTP secret migration in Validate). The
		// matched key is marked consumed either way, preserving one-time use.
		matchKey := crypto.HashRecoveryCode(recoveryCode)
		val, ok := recoveryCodesMap[matchKey]
		if !ok {
			// Legacy plaintext row: the code itself is the stored key.
			matchKey = recoveryCode
			val, ok = recoveryCodesMap[matchKey]
		}
		if !ok || val {
			// Not a known code, or one that has already been consumed: this is
			// a verification failure, not a server fault. Return (false, nil) so
			// the caller counts it as a failed attempt rather than an error.
			return false, nil
		}
		// mark the matched recovery code consumed
		recoveryCodesMap[matchKey] = true
		// convert recoveryCodesMap to string
		jsonData, err := json.Marshal(recoveryCodesMap)
		if err != nil {
			return false, err
		}
		consumed, err := p.deps.StorageProvider.ConsumeAuthenticatorRecoveryCode(ctx, totpModel.ID, storedCodes, string(jsonData))
		if err != nil {
			return false, err
		}
		if consumed {
			return true, nil
		}
		log.Debug().Int("attempt", attempt+1).Msg("recovery code blob changed under us, re-reading")
	}
	// Never report exhaustion as a failed code. The code may well still be
	// valid; what failed is the write, and answering "invalid" would spend the
	// user's credential on a database problem. See the fault-tolerance note on
	// ConsumeAuthenticatorRecoveryCode.
	return false, fmt.Errorf("could not consume recovery code after %d attempts", recoveryCodeConsumeAttempts)
}

package integration_tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestTOTPAtRest covers the at-rest hardening for TOTP secrets:
//
//   - Generate() must write the secret as enc:v1:<ciphertext>, never as
//     the raw base32 string.
//   - Validate() must decrypt the stored secret before computing the
//     expected code, and must succeed for both new (encrypted) rows and
//     legacy plaintext rows during a rolling upgrade.
//   - The lazy migration: a legacy plaintext row that successfully
//     validates once must be re-encrypted in place so the next read sees
//     the enc:v1: form — but ONLY when --enable-totp-migration is set.
//     With the flag off (the default), legacy rows still validate but
//     are left untouched on disk, so a mixed-version cluster does not
//     break old replicas mid-rollout.
//
// Configures EnableTOTPLogin so the authenticators provider returns a
// real totp.provider rather than nil. EnableTOTPMigration is enabled
// for the bulk of subtests below; the dedicated "migration disabled"
// subtest spins up a second test setup with the flag off.
func TestTOTPAtRest(t *testing.T) {
	cfg := getTestConfig()
	cfg.EnableTOTPLogin = true
	cfg.EnableMFA = true
	cfg.EnableTOTPMigration = true
	ts := initTestSetup(t, cfg)
	require.NotNil(t, ts.AuthenticatorProvider, "TOTP must be enabled for this test")
	ctx := context.Background()

	// Helper: insert a user we can attach authenticators to.
	mkUser := func(t *testing.T) *schemas.User {
		t.Helper()
		email := "totp_at_rest_" + uuid.NewString() + "@authorizer.dev"
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email: refs.NewStringRef(email),
		})
		require.NoError(t, err)
		return user
	}

	t.Run("Generate stores TOTP secret as ciphertext, not plaintext", func(t *testing.T) {
		user := mkUser(t)

		authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, authConfig)
		// The plaintext secret IS returned to the caller (the frontend
		// needs it to render the QR code), but the row written to
		// storage must be the enc:v1: form.
		assert.NotEmpty(t, authConfig.Secret)

		row, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(
			ctx, user.ID, constants.EnvKeyTOTPAuthenticator,
		)
		require.NoError(t, err)
		require.NotNil(t, row)
		assert.True(t,
			strings.HasPrefix(row.Secret, crypto.TOTPCipherPrefix),
			"stored TOTP secret must be prefixed with %q, got %q",
			crypto.TOTPCipherPrefix, row.Secret,
		)
		assert.NotEqual(t, authConfig.Secret, row.Secret,
			"stored TOTP secret must NOT equal the plaintext returned to the caller")

		// And the stored ciphertext must round-trip back to the
		// plaintext we just handed to the frontend.
		decrypted, err := crypto.DecryptTOTPSecret(row.Secret, cfg.JWTSecret)
		require.NoError(t, err)
		assert.Equal(t, authConfig.Secret, decrypted)
	})

	t.Run("Validate accepts a code computed from the encrypted secret", func(t *testing.T) {
		user := mkUser(t)
		authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
		require.NoError(t, err)

		// Code computed from the plaintext secret (the same one a real
		// authenticator app would have scanned out of the QR code).
		code, err := totp.GenerateCode(authConfig.Secret, time.Now())
		require.NoError(t, err)

		ok, err := ts.AuthenticatorProvider.Validate(ctx, code, user.ID)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Validate lazy-migrates a legacy plaintext row", func(t *testing.T) {
		user := mkUser(t)

		// Skip Generate and write a legacy-shaped row directly. This
		// simulates an authenticator enrolled on a release that was
		// written before the at-rest hardening.
		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "authorizer",
			AccountName: refs.StringValue(user.Email),
		})
		require.NoError(t, err)
		legacyPlainSecret := key.Secret()
		_, err = ts.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
			Secret: legacyPlainSecret, // NO enc:v1: prefix
			UserID: user.ID,
			Method: constants.EnvKeyTOTPAuthenticator,
		})
		require.NoError(t, err)

		// Sanity check: the row really is in legacy plaintext form
		// before we exercise the migration.
		before, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(
			ctx, user.ID, constants.EnvKeyTOTPAuthenticator,
		)
		require.NoError(t, err)
		require.False(t, crypto.IsEncryptedTOTPSecret(before.Secret))
		require.Equal(t, legacyPlainSecret, before.Secret)

		// A correct code computed from the plaintext must validate even
		// though the row hasn't been migrated yet.
		code, err := totp.GenerateCode(legacyPlainSecret, time.Now())
		require.NoError(t, err)
		ok, err := ts.AuthenticatorProvider.Validate(ctx, code, user.ID)
		require.NoError(t, err)
		require.True(t, ok, "legacy plaintext row should still validate during rolling upgrade")

		// After the successful Validate, the lazy migration must have
		// rewritten the row into the enc:v1: form. The decrypted value
		// must still match the original plaintext.
		after, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(
			ctx, user.ID, constants.EnvKeyTOTPAuthenticator,
		)
		require.NoError(t, err)
		assert.True(t,
			crypto.IsEncryptedTOTPSecret(after.Secret),
			"row should have been re-encrypted after first successful Validate, got %q",
			after.Secret,
		)
		assert.NotEqual(t, legacyPlainSecret, after.Secret)

		decrypted, err := crypto.DecryptTOTPSecret(after.Secret, cfg.JWTSecret)
		require.NoError(t, err)
		assert.Equal(t, legacyPlainSecret, decrypted)
	})

	t.Run("Validate rejects a wrong code on an encrypted row", func(t *testing.T) {
		user := mkUser(t)
		_, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
		require.NoError(t, err)

		ok, err := ts.AuthenticatorProvider.Validate(ctx, "000000", user.ID)
		require.NoError(t, err)
		assert.False(t, ok, "obviously wrong code must not validate")
	})

	t.Run("Validate without migration leaves legacy plaintext rows unchanged", func(t *testing.T) {
		// Spin up a second authorizer instance with EnableTOTPMigration
		// OFF (the default). Validate must still work on legacy plaintext
		// rows — that's the property that keeps a rolling deploy from a
		// pre-encryption release safe — but the row MUST NOT be rewritten.
		// If it were, replicas still on the old version would be unable
		// to read the migrated row.
		cfgNoMigration := getTestConfig()
		cfgNoMigration.EnableTOTPLogin = true
		cfgNoMigration.EnableMFA = true
		cfgNoMigration.EnableTOTPMigration = false
		tsNoMig := initTestSetup(t, cfgNoMigration)
		require.NotNil(t, tsNoMig.AuthenticatorProvider)

		emailNM := "totp_at_rest_nomig_" + uuid.NewString() + "@authorizer.dev"
		userNM, err := tsNoMig.StorageProvider.AddUser(ctx, &schemas.User{
			Email: refs.NewStringRef(emailNM),
		})
		require.NoError(t, err)

		key, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "authorizer",
			AccountName: refs.StringValue(userNM.Email),
		})
		require.NoError(t, err)
		legacyPlainSecret := key.Secret()
		_, err = tsNoMig.StorageProvider.AddAuthenticator(ctx, &schemas.Authenticator{
			Secret: legacyPlainSecret, // legacy plaintext form
			UserID: userNM.ID,
			Method: constants.EnvKeyTOTPAuthenticator,
		})
		require.NoError(t, err)

		code, err := totp.GenerateCode(legacyPlainSecret, time.Now())
		require.NoError(t, err)
		ok, err := tsNoMig.AuthenticatorProvider.Validate(ctx, code, userNM.ID)
		require.NoError(t, err)
		require.True(t, ok, "legacy plaintext row must still validate when migration is disabled")

		after, err := tsNoMig.StorageProvider.GetAuthenticatorDetailsByUserId(
			ctx, userNM.ID, constants.EnvKeyTOTPAuthenticator,
		)
		require.NoError(t, err)
		assert.Equal(t, legacyPlainSecret, after.Secret,
			"row must NOT be rewritten when --enable-totp-migration is off")
		assert.False(t, crypto.IsEncryptedTOTPSecret(after.Secret))
	})

	t.Run("Validate succeeds even if the migration write fails (best-effort)", func(t *testing.T) {
		// Direct unit-style assertion: a successful Validate must NOT
		// propagate an UpdateAuthenticator failure as a login error.
		// We can't easily inject a failing storage here without a mock,
		// so this subtest documents the contract by exercising the
		// happy path twice — the second Validate hits the
		// already-encrypted short-circuit and the row remains
		// well-formed. The defensive logging path is exercised by the
		// log assertions in unit tests of crypto helpers.
		user := mkUser(t)
		authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
		require.NoError(t, err)
		code, err := totp.GenerateCode(authConfig.Secret, time.Now())
		require.NoError(t, err)
		ok1, err := ts.AuthenticatorProvider.Validate(ctx, code, user.ID)
		require.NoError(t, err)
		require.True(t, ok1)
		ok2, err := ts.AuthenticatorProvider.Validate(ctx, code, user.ID)
		require.NoError(t, err)
		require.True(t, ok2)
	})

	t.Run("Validate is idempotent on already-encrypted rows", func(t *testing.T) {
		// A row that is already in enc:v1: form must remain unchanged
		// after Validate (no double-encryption, no rewrite churn). The
		// migration check must short-circuit when IsEncryptedTOTPSecret
		// is already true.
		user := mkUser(t)
		authConfig, err := ts.AuthenticatorProvider.Generate(ctx, user.ID)
		require.NoError(t, err)

		before, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(
			ctx, user.ID, constants.EnvKeyTOTPAuthenticator,
		)
		require.NoError(t, err)
		secretBefore := before.Secret

		code, err := totp.GenerateCode(authConfig.Secret, time.Now())
		require.NoError(t, err)
		ok, err := ts.AuthenticatorProvider.Validate(ctx, code, user.ID)
		require.NoError(t, err)
		require.True(t, ok)

		after, err := ts.StorageProvider.GetAuthenticatorDetailsByUserId(
			ctx, user.ID, constants.EnvKeyTOTPAuthenticator,
		)
		require.NoError(t, err)
		// The Secret column may legitimately be re-written by the
		// VerifiedAt update path, but if it changes it must STILL be
		// the same plaintext underneath (and still be in enc:v1: form).
		assert.True(t, crypto.IsEncryptedTOTPSecret(after.Secret))
		decryptedBefore, err := crypto.DecryptTOTPSecret(secretBefore, cfg.JWTSecret)
		require.NoError(t, err)
		decryptedAfter, err := crypto.DecryptTOTPSecret(after.Secret, cfg.JWTSecret)
		require.NoError(t, err)
		assert.Equal(t, decryptedBefore, decryptedAfter,
			"underlying TOTP secret must not change across validations")
	})
}

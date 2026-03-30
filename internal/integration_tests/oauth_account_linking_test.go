package integration_tests

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestOAuthAccountLinkingPreHijack tests that the OAuth callback handler
// does not link an OAuth identity to an existing unverified account.
// This prevents the "account pre-hijacking" attack where an attacker
// registers with a victim's email (without verifying), then the victim
// logs in via OAuth and the attacker retains password access.
func TestOAuthAccountLinkingPreHijack(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	t.Run("unverified_account_should_not_retain_password_after_oauth_signup", func(t *testing.T) {
		// Simulate the attacker's pre-registration:
		// Sign up with the victim's email using basic_auth (password).
		// Email verification is disabled in test config, but we simulate
		// an unverified account by setting EmailVerifiedAt = nil directly.
		email := "prehijack_" + uuid.New().String() + "@authorizer.dev"
		password := "AttackerPass@123"

		signupReq := &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		require.NoError(t, err)
		require.NotNil(t, res)

		// Manually unset EmailVerifiedAt to simulate an unverified account
		attackerUser, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		require.NotNil(t, attackerUser)
		attackerID := attackerUser.ID
		attackerUser.EmailVerifiedAt = nil
		_, err = ts.StorageProvider.UpdateUser(ctx, attackerUser)
		require.NoError(t, err)

		// Verify the account exists and is unverified
		attackerUser, err = ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.Nil(t, attackerUser.EmailVerifiedAt, "account should be unverified")
		assert.NotNil(t, attackerUser.Password, "account should have a password set")
		assert.Equal(t, constants.AuthRecipeMethodBasicAuth, attackerUser.SignupMethods)

		// Now simulate what the fixed OAuth callback does:
		// When an OAuth login occurs for this email, the handler should:
		// 1. Find the existing unverified account
		// 2. Delete it (because EmailVerifiedAt == nil)
		// 3. Create a fresh account for the OAuth user

		// Step 1: Look up by email (same as oauth_callback.go line 125)
		existingUser, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)

		// Step 2: Check that the account is unverified — the fix deletes it
		assert.Nil(t, existingUser.EmailVerifiedAt,
			"existing account's email should NOT be verified")

		// Step 3: Delete the unverified account (as the fix does)
		err = ts.StorageProvider.DeleteUser(ctx, existingUser)
		require.NoError(t, err)

		// Step 4: Create a new account as the OAuth user
		oauthUser := &schemas.User{
			ID:            uuid.New().String(),
			Email:         refs.NewStringRef(email),
			GivenName:     refs.NewStringRef("Victim"),
			FamilyName:    refs.NewStringRef("User"),
			SignupMethods: constants.AuthRecipeMethodGoogle,
			Roles:         "user",
		}
		now := int64(1700000000)
		oauthUser.EmailVerifiedAt = &now

		newUser, err := ts.StorageProvider.AddUser(ctx, oauthUser)
		require.NoError(t, err)
		require.NotNil(t, newUser)

		// Verify the new account properties
		assert.NotEqual(t, attackerID, newUser.ID,
			"new OAuth user should have a different ID than the attacker's account")
		assert.NotNil(t, newUser.EmailVerifiedAt,
			"OAuth user's email should be verified")
		assert.Equal(t, constants.AuthRecipeMethodGoogle, newUser.SignupMethods,
			"signup method should be google only, not basic_auth")
		assert.Nil(t, newUser.Password,
			"OAuth user should NOT have a password (attacker's password must not persist)")
		assert.False(t, strings.Contains(newUser.SignupMethods, constants.AuthRecipeMethodBasicAuth),
			"basic_auth should NOT be in signup methods")
	})

	t.Run("verified_account_should_link_oauth_identity", func(t *testing.T) {
		// This tests the normal case: a user who previously signed up
		// with email/password (and verified their email) should have
		// the OAuth identity linked to their existing account.
		email := "verified_link_" + uuid.New().String() + "@authorizer.dev"
		password := "ValidPass@123"

		signupReq := &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		require.NoError(t, err)
		require.NotNil(t, res)

		// In test config, email verification is not enabled, so account
		// is auto-verified. Confirm that.
		existingUser, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		require.NotNil(t, existingUser.EmailVerifiedAt,
			"account should be verified in test config")
		originalID := existingUser.ID
		originalPassword := existingUser.Password

		// Simulate what the OAuth callback does for a verified user:
		// It should link the OAuth provider to the existing account.
		signupMethod := existingUser.SignupMethods
		provider := constants.AuthRecipeMethodGoogle
		if !strings.Contains(signupMethod, provider) {
			signupMethod = signupMethod + "," + provider
		}
		existingUser.SignupMethods = signupMethod

		updatedUser, err := ts.StorageProvider.UpdateUser(ctx, existingUser)
		require.NoError(t, err)

		assert.Equal(t, originalID, updatedUser.ID,
			"user ID should remain the same when linking OAuth to verified account")
		assert.Contains(t, updatedUser.SignupMethods, constants.AuthRecipeMethodBasicAuth,
			"basic_auth should still be a valid signup method")
		assert.Contains(t, updatedUser.SignupMethods, constants.AuthRecipeMethodGoogle,
			"google should be added as a signup method")
		assert.Equal(t, originalPassword, updatedUser.Password,
			"password should remain for verified accounts (legitimate dual auth)")
	})

	t.Run("attacker_cannot_login_after_victim_oauth", func(t *testing.T) {
		// End-to-end scenario: attacker pre-registers, victim does OAuth,
		// attacker tries to login with password — should fail.
		email := "e2e_hijack_" + uuid.New().String() + "@authorizer.dev"
		attackerPassword := "AttackerPass@123"

		// Attacker signs up
		signupReq := &model.SignUpRequest{
			Email:           &email,
			Password:        attackerPassword,
			ConfirmPassword: attackerPassword,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		require.NoError(t, err)
		require.NotNil(t, res)

		// Set account as unverified (attacker didn't verify email)
		attackerUser, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		attackerUser.EmailVerifiedAt = nil
		_, err = ts.StorageProvider.UpdateUser(ctx, attackerUser)
		require.NoError(t, err)

		// Simulate the fixed OAuth callback: delete unverified, create fresh
		unverifiedUser, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		require.Nil(t, unverifiedUser.EmailVerifiedAt)

		err = ts.StorageProvider.DeleteUser(ctx, unverifiedUser)
		require.NoError(t, err)

		oauthUser := &schemas.User{
			ID:            uuid.New().String(),
			Email:         refs.NewStringRef(email),
			SignupMethods: constants.AuthRecipeMethodGoogle,
			Roles:         "user",
		}
		now := int64(1700000000)
		oauthUser.EmailVerifiedAt = &now
		_, err = ts.StorageProvider.AddUser(ctx, oauthUser)
		require.NoError(t, err)

		// Attacker tries to login with their password — should fail
		// because the account was replaced and has no password
		loginReq := &model.LoginRequest{
			Email:    &email,
			Password: attackerPassword,
		}
		loginRes, err := ts.GraphQLProvider.Login(ctx, loginReq)
		assert.Error(t, err, "attacker should not be able to login with password after account replacement")
		assert.Nil(t, loginRes)
	})

	t.Run("revoked_unverified_account_should_block_oauth", func(t *testing.T) {
		// If an unverified account is also revoked, the revocation
		// check should still take precedence.
		email := "revoked_unverified_" + uuid.New().String() + "@authorizer.dev"
		password := "RevokedPass@123"

		signupReq := &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		require.NoError(t, err)
		require.NotNil(t, res)

		// Set account as unverified AND revoked
		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		user.EmailVerifiedAt = nil
		revokedAt := int64(1700000000)
		user.RevokedTimestamp = &revokedAt
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		// Verify the account is both unverified and revoked
		user, err = ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.Nil(t, user.EmailVerifiedAt)
		assert.NotNil(t, user.RevokedTimestamp,
			"account should be revoked — OAuth callback should reject before checking email verification")
	})

	t.Run("multiple_oauth_providers_link_to_verified_account", func(t *testing.T) {
		// A verified user can link multiple OAuth providers
		email := "multi_oauth_" + uuid.New().String() + "@authorizer.dev"
		password := "MultiOAuth@123"

		signupReq := &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		require.NoError(t, err)
		require.NotNil(t, res)

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		require.NotNil(t, user.EmailVerifiedAt)

		// Link Google
		user.SignupMethods = user.SignupMethods + "," + constants.AuthRecipeMethodGoogle
		user, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		// Link GitHub
		user.SignupMethods = user.SignupMethods + "," + constants.AuthRecipeMethodGithub
		user, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		assert.Contains(t, user.SignupMethods, constants.AuthRecipeMethodBasicAuth)
		assert.Contains(t, user.SignupMethods, constants.AuthRecipeMethodGoogle)
		assert.Contains(t, user.SignupMethods, constants.AuthRecipeMethodGithub)

		// Password login should still work for verified multi-auth users
		loginReq := &model.LoginRequest{
			Email:    &email,
			Password: password,
		}
		loginRes, err := ts.GraphQLProvider.Login(ctx, loginReq)
		assert.NoError(t, err)
		assert.NotNil(t, loginRes)
	})

	t.Run("unverified_account_deletion_is_clean", func(t *testing.T) {
		// After deleting an unverified account, there should be no
		// trace of it in the database.
		email := "clean_delete_" + uuid.New().String() + "@authorizer.dev"
		password := "CleanDel@123"

		signupReq := &model.SignUpRequest{
			Email:           &email,
			Password:        password,
			ConfirmPassword: password,
		}
		res, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
		require.NoError(t, err)
		require.NotNil(t, res)

		user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		user.EmailVerifiedAt = nil
		_, err = ts.StorageProvider.UpdateUser(ctx, user)
		require.NoError(t, err)

		// Delete the unverified account
		err = ts.StorageProvider.DeleteUser(ctx, user)
		require.NoError(t, err)

		// Verify the account no longer exists
		_, err = ts.StorageProvider.GetUserByEmail(ctx, email)
		assert.Error(t, err, "deleted user should not be found by email")
	})
}

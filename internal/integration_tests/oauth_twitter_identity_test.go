package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// twitterSyntheticEmail mirrors internal/http_handlers/oauth_callback.go's
// unexported twitterSyntheticEmail helper (fmt.Sprintf("twitter-%s@twitter.oauth.internal", id)),
// used here to simulate what processTwitterUserInfo hands the
// OAuthCallbackHandler signup-vs-login branch for a given Twitter numeric id.
func twitterSyntheticEmail(twitterID string) string {
	return fmt.Sprintf("twitter-%s@twitter.oauth.internal", twitterID)
}

// TestOAuthTwitterIdentityStability is the regression test for the bug this
// change fixes: real Twitter never returns an email, so before this fix
// processTwitterUserInfo left User.Email nil and OAuthCallbackHandler's
// signup-vs-login lookup (GetUserByEmail(ctx, refs.StringValue(user.Email)))
// always ran as GetUserByEmail(ctx, "") — which never matches a NULL email
// column in SQL (NULL never equals an empty string) — so every single Twitter login
// created a brand-new, duplicate account.
//
// This test drives the exact same GetUserByEmail -> AddUser/UpdateUser
// sequence OAuthCallbackHandler runs (see oauth_account_linking_test.go for
// the established pattern of simulating the handler's storage-layer
// decisions directly rather than a full HTTP round trip), keyed on the
// synthetic email the fix now derives from Twitter's stable numeric id.
func TestOAuthTwitterIdentityStability(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	t.Run("same_twitter_id_resolves_to_same_user_on_second_login", func(t *testing.T) {
		twitterID := "tw-" + uuid.New().String()
		email := twitterSyntheticEmail(twitterID)

		// --- First login: OAuthCallbackHandler's GetUserByEmail misses, so it signs up. ---
		_, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.Error(t, err, "no account should exist yet for this Twitter id")

		now := time.Now().Unix()
		firstUser := &schemas.User{
			ID:            uuid.New().String(),
			Email:         refs.NewStringRef(email),
			GivenName:     refs.NewStringRef("Ada"),
			FamilyName:    refs.NewStringRef("Lovelace"),
			SignupMethods: constants.AuthRecipeMethodTwitter,
			Roles:         "user",
		}
		firstUser.EmailVerifiedAt = &now
		createdUser, err := ts.StorageProvider.AddUser(ctx, firstUser)
		require.NoError(t, err)
		require.NotNil(t, createdUser)

		// --- Second login with the SAME Twitter id: this is the bug's regression check. ---
		// Before the fix, Email was always nil for Twitter, so this lookup
		// would always be GetUserByEmail(ctx, "") and always miss, creating a
		// second, duplicate account. After the fix, the synthetic email is
		// stable across logins, so this must be a hit.
		existingUser, err := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, err, "second login with the same Twitter id must recognize the existing account")
		assert.Equal(t, createdUser.ID, existingUser.ID,
			"second login must resolve to the SAME user, not create a duplicate")

		// OAuthCallbackHandler's "existing user" branch updates in place
		// (UpdateUser), it never calls AddUser again.
		updatedUser, err := ts.StorageProvider.UpdateUser(ctx, existingUser)
		require.NoError(t, err)
		assert.Equal(t, createdUser.ID, updatedUser.ID)
	})

	t.Run("different_twitter_ids_never_collide_onto_the_same_user", func(t *testing.T) {
		idA := "tw-" + uuid.New().String()
		idB := "tw-" + uuid.New().String()
		emailA := twitterSyntheticEmail(idA)
		emailB := twitterSyntheticEmail(idB)
		require.NotEqual(t, emailA, emailB)

		now := time.Now().Unix()
		userA := &schemas.User{
			ID:            uuid.New().String(),
			Email:         refs.NewStringRef(emailA),
			SignupMethods: constants.AuthRecipeMethodTwitter,
			Roles:         "user",
		}
		userA.EmailVerifiedAt = &now
		createdA, err := ts.StorageProvider.AddUser(ctx, userA)
		require.NoError(t, err)

		// A GetUserByEmail lookup for the second (different) Twitter id must
		// miss - it must never accidentally resolve to the first user.
		_, err = ts.StorageProvider.GetUserByEmail(ctx, emailB)
		require.Error(t, err, "a different Twitter id must not match an unrelated existing account")

		userB := &schemas.User{
			ID:            uuid.New().String(),
			Email:         refs.NewStringRef(emailB),
			SignupMethods: constants.AuthRecipeMethodTwitter,
			Roles:         "user",
		}
		userB.EmailVerifiedAt = &now
		createdB, err := ts.StorageProvider.AddUser(ctx, userB)
		require.NoError(t, err)

		assert.NotEqual(t, createdA.ID, createdB.ID, "different Twitter ids must produce two distinct accounts")
	})
}

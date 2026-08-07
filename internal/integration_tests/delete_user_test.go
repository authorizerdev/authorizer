package integration_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestDeleteUser tests the delete user functionality by the admin
func TestDeleteUser(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "delete_user_test_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	// Signup the user
	signupReq := &model.SignUpRequest{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)
	require.NotNil(t, signupRes)
	require.NotNil(t, signupRes.User)
	// BREAKING: DeleteUser takes an id, not an email.
	userID := signupRes.User.ID

	t.Run("should fail without admin cookie", func(t *testing.T) {
		deleteRes, err := ts.GraphQLProvider.DeleteUser(ctx, &model.DeleteUserRequest{ID: userID})
		require.Error(t, err)
		require.Nil(t, deleteRes)
	})

	t.Run("should delete user", func(t *testing.T) {
		h, err := newAdminSessionToken(ts)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		deleteRes, err := ts.GraphQLProvider.DeleteUser(ctx, &model.DeleteUserRequest{ID: userID})
		require.NoError(t, err)
		require.NotNil(t, deleteRes)
	})
}

// TestDeleteUserByIDIncludingPhoneOnlyAccounts covers a reported gap: admin
// delete accepted only an email, so a phone-only account — which never gets one
// — could not be deleted at all. There was no second way in; the account was
// permanent.
//
// id is now the preferred identifier because it is the only one every account
// has, mirroring the id-or-email shape GetUserRequest already used. Email is
// kept for existing callers.
func TestDeleteUserByIDIncludingPhoneOnlyAccounts(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsSMSServiceEnabled = true
	cfg.EnableMobileBasicAuthentication = true
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)
	h, err := newAdminSessionToken(ts)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	t.Run("a phone-only account can be deleted by id", func(t *testing.T) {
		phone := fmt.Sprintf("+1%010d", time.Now().UnixNano()%10000000000)
		user, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			PhoneNumber:   &phone,
			SignupMethods: constants.AuthRecipeMethodMobileBasicAuth,
		})
		require.NoError(t, err)
		require.Empty(t, refs.StringValue(user.Email), "this account has no email — that is the point")

		res, err := ts.GraphQLProvider.DeleteUser(ctx, &model.DeleteUserRequest{ID: user.ID})
		require.NoError(t, err, "an account with no email must still be deletable")
		require.NotNil(t, res)

		_, err = ts.StorageProvider.GetUserByID(ctx, user.ID)
		assert.Error(t, err, "the account must actually be gone")
	})

	t.Run("an empty id is rejected", func(t *testing.T) {
		// The GraphQL schema marks id non-null, but a caller can still send "".
		// Without this guard that falls through to a lookup on the empty string,
		// whose result depends on the storage backend rather than on intent.
		_, err := ts.GraphQLProvider.DeleteUser(ctx, &model.DeleteUserRequest{ID: ""})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id is required")
	})
}

// TestDeleteUserDoesNotTouchOtherAccountsRows guards against cross-user data
// deletion in the post-delete cleanup.
//
// generateAndStoreOTP writes Email and PhoneNumber as plain strings, so an
// account holding only one of them stores the OTHER as "" — not NULL. The
// cleanup's lookups are `WHERE email = ?` / `WHERE phone_number = ?`, so
// passing "" matches every OTHER account in the same shape, and the row it
// returns is then DELETED.
//
// The scenario below is deterministic on purpose: the deleted account has no
// OTP of its own, so the empty-phone lookup can only match the bystander's.
// This half is reachable TODAY for email-only accounts — it does not need the
// phone-only delete this PR enables.
func TestDeleteUserDoesNotTouchOtherAccountsRows(t *testing.T) {
	cfg := getTestConfig()
	cfg.IsSMSServiceEnabled = true
	cfg.EnableMobileBasicAuthentication = true
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)
	h, err := newAdminSessionToken(ts)
	require.NoError(t, err)
	req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))

	mkEmailUser := func(t *testing.T) *schemas.User {
		t.Helper()
		email := "cross_delete_" + uuid.NewString() + "@authorizer.dev"
		u, err := ts.StorageProvider.AddUser(ctx, &schemas.User{
			Email:         &email,
			SignupMethods: constants.AuthRecipeMethodBasicAuth,
		})
		require.NoError(t, err)
		return u
	}

	// The bystander holds a live OTP. Its phone_number column is "" because the
	// account has no phone — that empty value is the collision key.
	bystander := mkEmailUser(t)
	_, err = ts.StorageProvider.UpsertOTP(ctx, &schemas.OTP{
		Email:       refs.StringValue(bystander.Email),
		PhoneNumber: refs.StringValue(bystander.PhoneNumber),
		Otp:         "123456",
		ExpiresAt:   time.Now().Add(5 * time.Minute).Unix(),
	})
	require.NoError(t, err)

	// The account actually being deleted has NO OTP, so an empty-phone lookup
	// cannot match its own row — only the bystander's.
	doomed := mkEmailUser(t)

	_, err = ts.GraphQLProvider.DeleteUser(ctx, &model.DeleteUserRequest{ID: doomed.ID})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, err := ts.StorageProvider.GetUserByID(ctx, doomed.ID)
		return err != nil
	}, 3*time.Second, 50*time.Millisecond, "the targeted account should be gone")
	// The cleanup is asynchronous; give it room to do damage if it is going to.
	time.Sleep(500 * time.Millisecond)

	otp, err := ts.StorageProvider.GetOTPByEmail(ctx, refs.StringValue(bystander.Email))
	require.NoError(t, err,
		"deleting an account with no phone number must not delete a DIFFERENT account's OTP via a `phone_number = \"\"` match")
	assert.NotNil(t, otp)
}

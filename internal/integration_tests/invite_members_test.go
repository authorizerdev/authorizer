package integration_tests

import (
	"fmt"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestInviteMembersUser tests the invite user functionality by the admin
func TestInviteMembersUser(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	req, ctx := createContext(ts)

	// Create a test user
	email := "invite_user_test_" + uuid.New().String() + "@authorizer.dev"
	emailTo := "test_user_invitation_" + uuid.New().String() + "@authorizer.dev"
	password := "Password@123"
	url := "https://authorizer.dev/"
	// Signup the user
	signupReq := &model.SignUpInput{
		Email:           &email,
		Password:        password,
		ConfirmPassword: password,
	}
	signupRes, err := ts.GraphQLProvider.SignUp(ctx, signupReq)
	require.NoError(t, err)
	require.NotNil(t, signupRes)
	require.NotNil(t, signupRes.User)

	t.Run("should fail without admin cookie", func(t *testing.T) {
		invitedUserDets, err := ts.GraphQLProvider.InviteMembers(ctx, &model.InviteMemberInput{
			Emails:      []string{emailTo},
			RedirectURI: &url,
		})
		require.Error(t, err)
		require.Nil(t, invitedUserDets)
	})

	t.Run("should fail to invite user as email sending is disabled", func(t *testing.T) {
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		_, err = ts.GraphQLProvider.InviteMembers(ctx, &model.InviteMemberInput{
			Emails:      []string{emailTo},
			RedirectURI: &url,
		})
		require.Error(t, err)
	})

	t.Run("should fail to invite user as email is blank", func(t *testing.T) {
		cfg.IsEmailServiceEnabled = true
		cfg.DisableBasicAuthentication = false
		cfg.DisableMagicLinkLogin = false
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		_, err = ts.GraphQLProvider.InviteMembers(ctx, &model.InviteMemberInput{
			Emails:      []string{},
			RedirectURI: &url,
		})
		require.Error(t, err)
	})

	t.Run("should invite user as email sending is enabled", func(t *testing.T) {
		cfg.IsEmailServiceEnabled = true
		cfg.DisableBasicAuthentication = false
		cfg.DisableMagicLinkLogin = false
		h, err := crypto.EncryptPassword(cfg.AdminSecret)
		assert.Nil(t, err)

		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", constants.AdminCookieName, h))
		invitedUserDets, err := ts.GraphQLProvider.InviteMembers(ctx, &model.InviteMemberInput{
			Emails:      []string{emailTo},
			RedirectURI: &url,
		})
		require.NoError(t, err)
		for _, user := range invitedUserDets.Users {
			assert.Equal(t, *user.Email, emailTo)
		}
	})
}

package integration_tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authorizerv1 "github.com/authorizerdev/authorizer/gen/go/authorizer/v1"
)

// TestAdminRevokeAccessGRPC exercises AuthorizerAdminService.RevokeAccess over
// gRPC: the fail-closed contract (no secret → Unauthenticated) and the happy
// path against a seeded user.
func TestAdminRevokeAccessGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	id, _ := seedUser(t, ts)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.RevokeAccess(context.Background(), &authorizerv1.RevokeAccessRequest{UserId: id})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("revokes access with admin secret", func(t *testing.T) {
		resp, err := client.RevokeAccess(adminCtx(cfg.AdminSecret), &authorizerv1.RevokeAccessRequest{UserId: id})
		require.NoError(t, err)
		require.Equal(t, "user access revoked successfully", resp.Message)
	})

	t.Run("revoking unknown user is an error", func(t *testing.T) {
		_, err := client.RevokeAccess(adminCtx(cfg.AdminSecret), &authorizerv1.RevokeAccessRequest{
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
	})
}

// TestAdminEnableAccessGRPC exercises AuthorizerAdminService.EnableAccess over
// gRPC: the fail-closed contract and the happy path against a seeded user that
// was previously revoked.
func TestAdminEnableAccessGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	id, _ := seedUser(t, ts)

	// revoke first so enabling has an effect
	_, err := client.RevokeAccess(adminCtx(cfg.AdminSecret), &authorizerv1.RevokeAccessRequest{UserId: id})
	require.NoError(t, err)

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.EnableAccess(context.Background(), &authorizerv1.EnableAccessRequest{UserId: id})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("enables access with admin secret", func(t *testing.T) {
		resp, err := client.EnableAccess(adminCtx(cfg.AdminSecret), &authorizerv1.EnableAccessRequest{UserId: id})
		require.NoError(t, err)
		require.Equal(t, "user access enabled successfully", resp.Message)
	})

	t.Run("enabling unknown user is an error", func(t *testing.T) {
		_, err := client.EnableAccess(adminCtx(cfg.AdminSecret), &authorizerv1.EnableAccessRequest{
			UserId: uuid.New().String(),
		})
		require.Error(t, err)
	})
}

// TestAdminInviteMembersGRPC exercises AuthorizerAdminService.InviteMembers over
// gRPC: the fail-closed contract and the happy path. The email/auth feature
// flags are enabled on the shared config pointer before invoking (matching the
// existing GraphQL invite_members test pattern).
func TestAdminInviteMembersGRPC(t *testing.T) {
	client, ts := newAdminClientWithSetup(t)
	cfg := ts.Config
	cfg.IsEmailServiceEnabled = true
	cfg.EnableBasicAuthentication = true
	cfg.EnableMagicLinkLogin = true

	email := "admin-invite-grpc-" + uuid.New().String() + "@authorizer.test"

	t.Run("fail closed without admin secret", func(t *testing.T) {
		_, err := client.InviteMembers(context.Background(), &authorizerv1.InviteMembersRequest{
			Emails: []string{email},
		})
		require.Error(t, err)
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("no valid emails is an error", func(t *testing.T) {
		_, err := client.InviteMembers(adminCtx(cfg.AdminSecret), &authorizerv1.InviteMembersRequest{
			Emails: []string{"not-an-email"},
		})
		require.Error(t, err)
	})

	t.Run("invites new member with admin secret", func(t *testing.T) {
		resp, err := client.InviteMembers(adminCtx(cfg.AdminSecret), &authorizerv1.InviteMembersRequest{
			Emails: []string{email},
		})
		require.NoError(t, err)
		require.Len(t, resp.Users, 1)
		require.Equal(t, email, resp.Users[0].Email)
	})
}

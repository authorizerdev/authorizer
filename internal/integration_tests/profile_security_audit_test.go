package integration_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
)

// A password change and an MFA disable are security events. Both were folded
// into AuditProfileUpdatedEvent, indistinguishable from a display-name edit —
// while AuditPasswordChangedEvent and AuditMFADisabledEvent sat declared and
// never emitted. Anyone auditing "who changed a password" had nothing to query.
func TestProfileSecurityEventsAreAuditedSeparately(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)
	_, ctx := createContext(ts)

	email := "profile_audit_" + uuid.NewString() + "@authorizer.dev"
	const password = "Password@123"
	_, err := ts.GraphQLProvider.SignUp(ctx, &model.SignUpRequest{
		Email: &email, Password: password, ConfirmPassword: password,
	})
	require.NoError(t, err)

	loginRes, err := ts.GraphQLProvider.Login(ctx, &model.LoginRequest{
		Email: &email, Password: password,
	})
	require.NoError(t, err)
	require.NotNil(t, loginRes.AccessToken)
	ts.GinContext.Request.Header.Set("Authorization", "Bearer "+*loginRes.AccessToken)

	awaitAudit := func(t *testing.T, action, userID string) {
		t.Helper()
		require.Eventually(t, func() bool {
			logs, _, lErr := ts.StorageProvider.ListAuditLogs(ctx,
				&model.Pagination{Limit: 50, Page: 1},
				map[string]interface{}{"action": action})
			if lErr != nil {
				return false
			}
			for _, l := range logs {
				if l.ActorID == userID {
					return true
				}
			}
			return false
		}, 5*time.Second, 25*time.Millisecond,
			"no %s audit entry for the user that performed it", action)
	}

	user, err := ts.StorageProvider.GetUserByEmail(ctx, email)
	require.NoError(t, err)

	t.Run("a password change is recorded as its own event", func(t *testing.T) {
		newPassword := "Password@1234"
		_, err := ts.GraphQLProvider.UpdateProfile(ctx, &model.UpdateProfileRequest{
			OldPassword:        refs.NewStringRef(password),
			NewPassword:        refs.NewStringRef(newPassword),
			ConfirmNewPassword: refs.NewStringRef(newPassword),
		})
		require.NoError(t, err)

		awaitAudit(t, constants.AuditPasswordChangedEvent, user.ID)
		// The generic event still fires, so nothing consuming it today breaks.
		awaitAudit(t, constants.AuditProfileUpdatedEvent, user.ID)
	})

	t.Run("disabling MFA is recorded as its own event", func(t *testing.T) {
		// Enable directly in storage: the enable path has its own gating, and the
		// event under test is the DISABLE.
		u, gErr := ts.StorageProvider.GetUserByEmail(ctx, email)
		require.NoError(t, gErr)
		u.IsMultiFactorAuthEnabled = refs.NewBoolRef(true)
		_, uErr := ts.StorageProvider.UpdateUser(ctx, u)
		require.NoError(t, uErr)

		_, err := ts.GraphQLProvider.UpdateProfile(ctx, &model.UpdateProfileRequest{
			IsMultiFactorAuthEnabled: refs.NewBoolRef(false),
		})
		require.NoError(t, err)

		awaitAudit(t, constants.AuditMFADisabledEvent, user.ID)
	})
}

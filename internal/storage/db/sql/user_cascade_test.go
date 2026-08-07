package sql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestUserOwnedModelsMatchCollections keeps the GORM model list behind the
// DeleteUser cascade in lockstep with schemas.UserOwnedCollections, which the
// other five backends iterate directly. Without this, adding a user-keyed table
// to the shared list would cascade on five backends and silently skip SQL.
func TestUserOwnedModelsMatchCollections(t *testing.T) {
	cfg := sqlMigrationTestConfig(t, "sqlite")
	p, err := NewProvider(cfg, sqlTestDeps(t))
	require.NoError(t, err)
	defer func() { _ = p.Close() }()

	got := make([]string, 0, len(userOwnedModels))
	for _, m := range userOwnedModels {
		stmt := &gorm.Statement{DB: p.db}
		require.NoError(t, stmt.Parse(m))
		got = append(got, stmt.Table)
	}
	assert.ElementsMatch(t, schemas.UserOwnedCollections, got)
}

// TestDeleteUserCascadesAllUserOwnedRows proves a hard delete removes every row
// keyed on the user id, not just their sessions. The federated-identity row is
// the one that matters most: left behind it points at a dead user id, SSO login
// fails closed on it, and the unique (org_id, issuer, subject) triple blocks
// re-provisioning — a permanent lockout.
func TestDeleteUserCascadesAllUserOwnedRows(t *testing.T) {
	for _, dbType := range sqlMigrationTestDBTypes() {
		t.Run(dbType, func(t *testing.T) {
			cfg := sqlMigrationTestConfig(t, dbType)
			p, err := NewProvider(cfg, sqlTestDeps(t))
			require.NoError(t, err)
			defer func() { _ = p.Close() }()

			ctx := context.Background()
			user, err := p.AddUser(ctx, &schemas.User{
				Email:         refs.NewStringRef("cascade-" + dbType + "@example.com"),
				SignupMethods: "basic_auth",
			})
			require.NoError(t, err)

			const orgID = "org-cascade"
			const issuer = "https://idp.example.com"
			const subject = "upstream-subject-1"

			require.NoError(t, p.AddSession(ctx, &schemas.Session{UserID: user.ID}))
			_, err = p.AddFederatedIdentity(ctx, &schemas.FederatedIdentity{
				OrgID: orgID, Issuer: issuer, Subject: subject, UserID: user.ID,
			})
			require.NoError(t, err)
			_, err = p.AddOrgMembership(ctx, &schemas.OrgMembership{OrgID: orgID, UserID: user.ID, Roles: "member"})
			require.NoError(t, err)
			_, err = p.AddAuthenticator(ctx, &schemas.Authenticator{UserID: user.ID, Method: "totp", Secret: "s3cret"})
			require.NoError(t, err)
			_, err = p.AddWebauthnCredential(ctx, &schemas.WebauthnCredential{
				UserID: user.ID, CredentialID: "cred-" + dbType, PublicKey: "pk", Name: "laptop",
			})
			require.NoError(t, err)
			require.NoError(t, p.AddSessionToken(ctx, &schemas.SessionToken{UserID: user.ID, KeyName: "access", Token: "t"}))
			require.NoError(t, p.AddMFASession(ctx, &schemas.MFASession{UserID: user.ID, KeyName: "mfa"}))

			require.NoError(t, p.DeleteUser(ctx, user))

			// Every user-keyed table must be empty for this id.
			for _, m := range userOwnedModels {
				var count int64
				require.NoError(t, p.db.WithContext(ctx).Model(m).Where("user_id = ?", user.ID).Count(&count).Error)
				stmt := &gorm.Statement{DB: p.db}
				require.NoError(t, stmt.Parse(m))
				assert.Zero(t, count, "%s still holds rows for the deleted user", stmt.Table)
			}

			// Lockout regression: the triple is free again, so the same upstream
			// principal can be re-provisioned instead of failing closed forever.
			_, err = p.GetFederatedIdentity(ctx, orgID, issuer, subject)
			require.Error(t, err, "federated identity must be gone after the user is deleted")
			fresh, err := p.AddUser(ctx, &schemas.User{
				Email:         refs.NewStringRef("cascade-reprovision-" + dbType + "@example.com"),
				SignupMethods: "basic_auth",
			})
			require.NoError(t, err)
			_, err = p.AddFederatedIdentity(ctx, &schemas.FederatedIdentity{
				OrgID: orgID, Issuer: issuer, Subject: subject, UserID: fresh.ID,
			})
			require.NoError(t, err, "re-provisioning the same (org, issuer, subject) must succeed")
		})
	}
}

// TestDeleteUserAtomicRollback proves the whole cascade is one transaction: when
// a later step fails, the user row is not left deleted on its own. Without it a
// partial cascade would strand exactly the orphans this test suite is about.
func TestDeleteUserAtomicRollback(t *testing.T) {
	cfg := sqlMigrationTestConfig(t, "sqlite")
	p, err := NewProvider(cfg, sqlTestDeps(t))
	require.NoError(t, err)
	defer func() { _ = p.Close() }()

	ctx := context.Background()
	user, err := p.AddUser(ctx, &schemas.User{
		Email:         refs.NewStringRef("rollback@example.com"),
		SignupMethods: "basic_auth",
	})
	require.NoError(t, err)

	// Force a cascade step to fail deterministically.
	require.NoError(t, p.db.Migrator().DropTable(&schemas.MFASession{}))

	require.Error(t, p.DeleteUser(ctx, user), "delete must fail when a cascade step errors")

	got, err := p.GetUserByID(ctx, user.ID)
	require.NoError(t, err, "user must still exist after the cascade rolled back")
	assert.Equal(t, user.ID, got.ID)
}

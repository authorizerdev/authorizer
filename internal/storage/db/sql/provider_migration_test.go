package sql

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

func sqlTestDeps(t *testing.T) *Dependencies {
	t.Helper()
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	return &Dependencies{Log: &logger}
}

// sqlMigrationTestDBTypes returns the SQL backends to exercise. Honors TEST_DBS
// (comma-separated); defaults to sqlite so local runs stay light. CI sets
// TEST_DBS=postgres to cover the engine where the original failure was reported.
func sqlMigrationTestDBTypes() []string {
	supported := map[string]bool{
		constants.DbTypePostgres: true,
		constants.DbTypeSqlite:   true,
		constants.DbTypeMysql:    true,
		constants.DbTypeMariaDB:  true,
	}
	env := os.Getenv("TEST_DBS")
	if env == "" {
		return []string{constants.DbTypeSqlite}
	}
	var out []string
	for _, p := range strings.Split(env, ",") {
		p = strings.TrimSpace(p)
		if supported[p] {
			out = append(out, p)
		}
	}
	return out
}

// sqlMigrationTestConfig builds a config for dbType, skipping the test when the
// backend is not reachable in this environment.
func sqlMigrationTestConfig(t *testing.T, dbType string) *config.Config {
	t.Helper()
	cfg := &config.Config{
		DatabaseType: dbType,
		DatabaseName: "authorizer_test",
		DefaultRoles: []string{"user"},
	}
	switch dbType {
	case constants.DbTypeSqlite:
		cfg.DatabaseURL = filepath.Join(t.TempDir(), "migration_test.db")
	case constants.DbTypePostgres:
		cfg.DatabaseURL = "postgres://postgres:postgres@localhost:5434/postgres"
		skipIfTCPClosed(t, "localhost:5434")
	case constants.DbTypeMysql, constants.DbTypeMariaDB:
		cfg.DatabaseURL = "root:password@tcp(localhost:3306)/authorizer_test"
		skipIfTCPClosed(t, "localhost:3306")
	default:
		t.Skipf("unsupported SQL db type for migration test: %s", dbType)
	}
	return cfg
}

func skipIfTCPClosed(t *testing.T, addr string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Skipf("skipping: %s not reachable: %v", addr, err)
	}
	_ = conn.Close()
}

// TestProviderEmailPhoneUpdatesAndUniqueness covers v2's application-layer
// uniqueness for users and OTPs (email and phone_number are non-unique indexes
// in the DB). It asserts that a user can update their own email/phone number
// seamlessly, that duplicates across different users are still rejected, and
// that OTPs key independently on email and phone.
func TestProviderEmailPhoneUpdatesAndUniqueness(t *testing.T) {
	for _, dbType := range sqlMigrationTestDBTypes() {
		t.Run(dbType, func(t *testing.T) {
			cfg := sqlMigrationTestConfig(t, dbType)
			p, err := NewProvider(cfg, sqlTestDeps(t))
			require.NoError(t, err)
			require.NotNil(t, p)
			ctx := context.Background()

			uniq := strings.ReplaceAll(uuid.New().String(), "-", "")
			emailA := "a_" + uniq + "@test.com"
			phoneA := "+1100" + uniq[:9]

			userA := &schemas.User{
				ID:            uuid.New().String(),
				Email:         refs.NewStringRef(emailA),
				PhoneNumber:   refs.NewStringRef(phoneA),
				SignupMethods: "basic_auth",
			}
			userA, err = p.AddUser(ctx, userA)
			require.NoError(t, err)

			// Seamless self-update: change A's phone number to a new value.
			newPhoneA := "+1900" + uniq[:9]
			userA.PhoneNumber = refs.NewStringRef(newPhoneA)
			_, err = p.UpdateUser(ctx, userA)
			require.NoError(t, err, "updating own phone number should succeed")
			gotByPhone, err := p.GetUserByPhoneNumber(ctx, newPhoneA)
			require.NoError(t, err)
			assert.Equal(t, userA.ID, gotByPhone.ID)

			// Seamless self-update: change A's email, then re-save (idempotent).
			newEmailA := "a2_" + uniq + "@test.com"
			userA.Email = refs.NewStringRef(newEmailA)
			_, err = p.UpdateUser(ctx, userA)
			require.NoError(t, err, "updating own email should succeed")
			_, err = p.UpdateUser(ctx, userA)
			require.NoError(t, err, "re-saving the same user should succeed")
			gotByEmail, err := p.GetUserByEmail(ctx, newEmailA)
			require.NoError(t, err)
			assert.Equal(t, userA.ID, gotByEmail.ID)

			// A different user cannot claim A's email or phone via AddUser.
			_, err = p.AddUser(ctx, &schemas.User{
				ID: uuid.New().String(), Email: refs.NewStringRef(newEmailA), SignupMethods: "basic_auth",
			})
			assert.Error(t, err, "duplicate email should be rejected")
			_, err = p.AddUser(ctx, &schemas.User{
				ID: uuid.New().String(), PhoneNumber: refs.NewStringRef(newPhoneA), SignupMethods: "basic_auth",
			})
			assert.Error(t, err, "duplicate phone should be rejected")

			// OTPs key on email and phone independently; upsert by the same email
			// updates in place.
			otpEmail := &schemas.OTP{Email: newEmailA, Otp: "111111", ExpiresAt: time.Now().Add(5 * time.Minute).Unix()}
			_, err = p.UpsertOTP(ctx, otpEmail)
			require.NoError(t, err)
			otpEmail.Otp = "222222"
			_, err = p.UpsertOTP(ctx, otpEmail)
			require.NoError(t, err)
			fetchedOTP, err := p.GetOTPByEmail(ctx, newEmailA)
			require.NoError(t, err)
			assert.Equal(t, "222222", fetchedOTP.Otp)

			otpPhone := &schemas.OTP{PhoneNumber: newPhoneA, Otp: "333333", ExpiresAt: time.Now().Add(5 * time.Minute).Unix()}
			_, err = p.UpsertOTP(ctx, otpPhone)
			require.NoError(t, err)
			fetchedPhoneOTP, err := p.GetOTPByPhoneNumber(ctx, newPhoneA)
			require.NoError(t, err)
			assert.Equal(t, "333333", fetchedPhoneOTP.Otp)

			// Cleanup.
			_ = p.DeleteOTP(ctx, fetchedOTP)
			_ = p.DeleteOTP(ctx, fetchedPhoneOTP)
			_ = p.DeleteUser(ctx, userA)
		})
	}
}

// TestStaleUniqueConstraintMigration reproduces an upgraded v1 database whose
// email/phone columns still carry UNIQUE objects, and asserts the provider
// clears them on startup instead of aborting with SQLSTATE 42704
// (regression introduced by the GORM 1.25.10 bump in 2.3.0-rc.1). Postgres only:
// it has named UNIQUE constraints and standalone unique indexes with stable
// DDL; sqlite rebuilds tables and has no comparable failure mode.
func TestStaleUniqueConstraintMigration(t *testing.T) {
	for _, dbType := range sqlMigrationTestDBTypes() {
		if dbType != constants.DbTypePostgres {
			continue
		}
		t.Run(dbType, func(t *testing.T) {
			cfg := sqlMigrationTestConfig(t, dbType)
			deps := sqlTestDeps(t)
			ctx := context.Background()

			// First boot creates the v2 schema (non-unique indexes).
			p1, err := NewProvider(cfg, deps)
			require.NoError(t, err)

			// Simulate a v1 database, covering every real-world stale form so the
			// name-agnostic cleanup is exercised end-to-end:
			//   - users.email:         UNIQUE CONSTRAINT "authorizer_users_email_key"
			//                          (Postgres default for a gorm:"unique" tag)
			//   - users.phone_number:  UNIQUE CONSTRAINT "my_legacy_phone_uq"
			//                          (arbitrary/custom name — only a catalog-driven
			//                          drop, not name enumeration, can catch this)
			//   - otps.phone_number:   UNIQUE CONSTRAINT "idx_authorizer_otps_phone_number"
			//                          (idx_-named constraint; backing index can NOT
			//                          be dropped with DROP INDEX — field-reported case)
			//   - otps.email:          standalone UNIQUE INDEX "idx_authorizer_otps_email"
			// Idempotent so reruns against the shared test DB don't clash.
			for _, stmt := range []string{
				`ALTER TABLE authorizer_users DROP CONSTRAINT IF EXISTS authorizer_users_email_key`,
				`ALTER TABLE authorizer_users ADD CONSTRAINT authorizer_users_email_key UNIQUE (email)`,
				`ALTER TABLE authorizer_users DROP CONSTRAINT IF EXISTS my_legacy_phone_uq`,
				`ALTER TABLE authorizer_users ADD CONSTRAINT my_legacy_phone_uq UNIQUE (phone_number)`,
				`ALTER TABLE authorizer_otps DROP CONSTRAINT IF EXISTS idx_authorizer_otps_phone_number`,
				`DROP INDEX IF EXISTS idx_authorizer_otps_phone_number`,
				`ALTER TABLE authorizer_otps ADD CONSTRAINT idx_authorizer_otps_phone_number UNIQUE (phone_number)`,
				`DROP INDEX IF EXISTS idx_authorizer_otps_email`,
				`CREATE UNIQUE INDEX idx_authorizer_otps_email ON authorizer_otps (email)`,
			} {
				require.NoError(t, p1.db.Exec(stmt).Error, stmt)
			}
			require.True(t, p1.db.Migrator().HasConstraint(&schemas.User{}, "authorizer_users_email_key"),
				"precondition: stale unique constraint seeded")
			require.True(t, p1.db.Migrator().HasConstraint(&schemas.User{}, "my_legacy_phone_uq"),
				"precondition: custom-named unique constraint seeded")
			require.True(t, p1.db.Migrator().HasConstraint(&schemas.OTP{}, "idx_authorizer_otps_phone_number"),
				"precondition: idx_-named unique constraint seeded")

			// Second boot must clear the stale uniqueness and complete migration.
			p2, err := NewProvider(cfg, deps)
			require.NoError(t, err, "startup must not abort with SQLSTATE 42704")

			assert.False(t, p2.db.Migrator().HasConstraint(&schemas.User{}, "authorizer_users_email_key"),
				"stale unique constraint should be dropped")
			assert.False(t, p2.db.Migrator().HasConstraint(&schemas.User{}, "my_legacy_phone_uq"),
				"custom-named unique constraint should be dropped")
			assert.False(t, p2.db.Migrator().HasConstraint(&schemas.OTP{}, "idx_authorizer_otps_phone_number"),
				"stale idx_-named unique constraint should be dropped")

			// The search index is preserved, just no longer unique: AutoMigrate
			// recreates idx_authorizer_otps_phone_number as a non-unique index.
			indexes, err := p2.db.Migrator().GetIndexes(&schemas.OTP{})
			require.NoError(t, err)
			var phoneIdx string
			for _, idx := range indexes {
				if cols := idx.Columns(); len(cols) == 1 && cols[0] == "phone_number" {
					phoneIdx = idx.Name()
					if unique, ok := idx.Unique(); ok {
						assert.False(t, unique, "otps.phone_number index should no longer be unique")
					}
				}
			}
			assert.NotEmpty(t, phoneIdx, "non-unique search index on otps.phone_number should still exist")

			// After migrating a legacy DB, a user can still update their phone
			// number seamlessly (no stale constraint in the way).
			uniq := strings.ReplaceAll(uuid.New().String(), "-", "")
			u := &schemas.User{
				ID:            uuid.New().String(),
				Email:         refs.NewStringRef("m_" + uniq + "@test.com"),
				PhoneNumber:   refs.NewStringRef("+1700" + uniq[:9]),
				SignupMethods: "basic_auth",
			}
			u, err = p2.AddUser(ctx, u)
			require.NoError(t, err)
			u.PhoneNumber = refs.NewStringRef("+1600" + uniq[:9])
			_, err = p2.UpdateUser(ctx, u)
			require.NoError(t, err, "updating phone after migration should succeed")
			_ = p2.DeleteUser(ctx, u)
		})
	}
}

// TestOAuthStateTableName asserts the SQL OAuthState table is named
// authorizer_oauth_states, not the authorizer_o_auth_states GORM's naming
// strategy would otherwise derive from the struct name (it splits "OAuth"
// into "O"+"Auth"). Every other storage provider already uses
// authorizer_oauth_states; schemas.OAuthState.TableName() keeps SQL in sync.
func TestOAuthStateTableName(t *testing.T) {
	for _, dbType := range sqlMigrationTestDBTypes() {
		t.Run(dbType, func(t *testing.T) {
			cfg := sqlMigrationTestConfig(t, dbType)
			p, err := NewProvider(cfg, sqlTestDeps(t))
			require.NoError(t, err)

			assert.True(t, p.db.Migrator().HasTable(schemas.Collections.OAuthState),
				"authorizer_oauth_states should exist")
			assert.False(t, p.db.Migrator().HasTable(schemas.Prefix+"o_auth_states"),
				"mangled authorizer_o_auth_states should not exist")
		})
	}
}

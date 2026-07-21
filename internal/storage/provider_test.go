package storage

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allDBTypes is the full list of database types supported for storage tests.
var allDBTypes = []string{
	constants.DbTypePostgres,
	constants.DbTypeSqlite,
	constants.DbTypeMongoDB,
	constants.DbTypeArangoDB,
	constants.DbTypeScyllaDB,
	constants.DbTypeCouchbaseDB,
	constants.DbTypeDynamoDB,
}

// getTestDBTypes returns the list of database types to test against.
// Reads from TEST_DBS env var (comma-separated). Defaults to allDBTypes if not set.
func getTestDBTypes() []string {
	testDBsEnv := os.Getenv("TEST_DBS")
	if testDBsEnv == "" {
		return allDBTypes
	}

	parts := strings.Split(testDBsEnv, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func getTestDBConfig(dbType string) *config.Config {
	cfg := &config.Config{
		DatabaseName: "authorizer_test",
		AWSRegion:    "us-east-1",
	}
	cfg.DatabaseType = dbType

	// Set specific database URLs based on type
	switch dbType {
	case constants.DbTypePostgres:
		cfg.DatabaseURL = "postgres://postgres:postgres@localhost:5434/postgres"
	case constants.DbTypeSqlite:
		cfg.DatabaseURL = "test.db"
	case constants.DbTypeMongoDB:
		cfg.DatabaseURL = "mongodb://localhost:27017"
	case constants.DbTypeArangoDB:
		cfg.DatabaseURL = "http://localhost:8529"
	case constants.DbTypeScyllaDB:
		cfg.DatabaseURL = "127.0.0.1:9042"
	case constants.DbTypeCouchbaseDB:
		cfg.DatabaseURL = "couchbase://127.0.0.1"
		// Allow extra time for Couchbase container to become ready in tests (test-all-db runs Couchbase last)
		cfg.CouchBaseWaitTimeout = 300
	case constants.DbTypeDynamoDB:
		// Must be a client-routable host (not bind address 0.0.0.0); matches integration_tests getDBURL.
		cfg.DatabaseURL = "http://127.0.0.1:8000"
	}

	return cfg
}

func TestStorageProvider(t *testing.T) {
	// Initialize logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	for _, dbType := range getTestDBTypes() {
		t.Run("should test storage provider for "+dbType, func(t *testing.T) {
			if dbType == constants.DbTypeDynamoDB {
				_ = os.Unsetenv("AWS_ACCESS_KEY_ID")
				_ = os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			}
			cfg := getTestDBConfig(dbType)
			if dbType == constants.DbTypeCouchbaseDB {
				// Skip Couchbase tests quickly if the local Couchbase instance is not reachable.
				// This avoids long WaitUntilReady timeouts when the container is not running.
				conn, err := net.DialTimeout("tcp", "127.0.0.1:8091", 2*time.Second)
				if err != nil {
					t.Skipf("Skipping Couchbase storage tests: Couchbase not reachable on 127.0.0.1:8091: %v", err)
				}
				_ = conn.Close()

				cfg.DatabaseUsername = "Administrator"
				cfg.DatabasePassword = "password"
			}
			ctx := context.Background()
			var provider Provider
			var err error
			if dbType == constants.DbTypeCouchbaseDB {
				for attempt := 1; attempt <= 3; attempt++ {
					provider, err = New(cfg, &Dependencies{
						Log: &logger,
					})
					if err == nil {
						break
					}
					if attempt < 3 {
						t.Logf("Couchbase provider attempt %d failed: %v; retrying...", attempt, err)
						time.Sleep(5 * time.Second)
					}
				}
			} else {
				provider, err = New(cfg, &Dependencies{
					Log: &logger,
				})
			}
			if err != nil {
				log.Error().Err(err).Msg("failed to create storage provider")
			}
			require.NoError(t, err)
			require.NotNil(t, provider)

			t.Run("HealthCheck", func(t *testing.T) {
				err := provider.HealthCheck(ctx)
				assert.NoError(t, err, "HealthCheck should succeed when the test database is reachable")
			})

			t.Run("Authenticator Operations", func(t *testing.T) {
				testAuthenticatorOperations(t, ctx, provider, dbType)
			})

			t.Run("Email Template Operations", func(t *testing.T) {
				testEmailTemplateOperations(t, ctx, provider, dbType)
			})

			t.Run("OTP Operations", func(t *testing.T) {
				testOTPOperations(t, ctx, provider)
			})

			t.Run("Session Operations", func(t *testing.T) {
				testSessionOperations(t, ctx, provider)
			})

			t.Run("User Operations", func(t *testing.T) {
				testUserOperations(t, ctx, provider, dbType)
			})

			t.Run("User Search Operations", func(t *testing.T) {
				testUserSearchOperations(t, ctx, provider)
			})

			t.Run("Verification Request Operations", func(t *testing.T) {
				testVerificationRequestOperations(t, ctx, provider, dbType)
			})

			t.Run("Webhook Operations", func(t *testing.T) {
				testWebhookOperations(t, ctx, provider, dbType)
			})

			t.Run("Session Token Operations", func(t *testing.T) {
				testSessionTokenOperations(t, ctx, provider)
			})

			t.Run("MFA Session Operations", func(t *testing.T) {
				testMFASessionOperations(t, ctx, provider)
			})

			t.Run("OAuth State Operations", func(t *testing.T) {
				testOAuthStateOperations(t, ctx, provider)
			})

			t.Run("Audit Log Operations", func(t *testing.T) {
				testAuditLogOperations(t, ctx, provider)
			})

			t.Run("Client Operations", func(t *testing.T) {
				testClientOperations(t, ctx, provider)
			})

			t.Run("Trusted Issuer Operations", func(t *testing.T) {
				testTrustedIssuerOperations(t, ctx, provider, dbType)
			})

			t.Run("Organization Operations", func(t *testing.T) {
				testOrganizationOperations(t, ctx, provider)
			})

			t.Run("Org Membership Operations", func(t *testing.T) {
				testOrgMembershipOperations(t, ctx, provider)
			})

			t.Run("Org OIDC Connection & Federated Identity Operations", func(t *testing.T) {
				testOrgOIDCAndFederatedOperations(t, ctx, provider, dbType)
			})

			t.Run("Org SAML Connection Operations", func(t *testing.T) {
				testOrgSAMLOperations(t, ctx, provider, dbType)
			})

			t.Run("SAML IdP Storage Operations", func(t *testing.T) {
				testSAMLIDPStorageOperations(t, ctx, provider)
			})

			t.Run("SCIM Endpoint Operations", func(t *testing.T) {
				testScimEndpointOperations(t, ctx, provider)
			})
			t.Run("SCIM Group Operations", func(t *testing.T) {
				testScimGroupOperations(t, ctx, provider)
			})
			t.Run("Org Domain Operations", func(t *testing.T) {
				testOrgDomainOperations(t, ctx, provider)
			})

			t.Run("User SCIM Fields", func(t *testing.T) {
				testUserScimFields(t, ctx, provider)
			})

			if isSQLTestDB(dbType) {
				t.Run("SQL CRUD Correctness Fixes", func(t *testing.T) {
					testSQLCRUDCorrectnessFixes(t, ctx, provider)
				})
			}

		})
	}
}

// isSQLTestDB reports whether dbType is backed by the shared GORM SQL provider.
// The CRUD-correctness regression tests below assert behaviour specific to that
// provider (uniqueness pre-check, LIKE anchoring, partial-struct guard).
func isSQLTestDB(dbType string) bool {
	switch dbType {
	case constants.DbTypePostgres, constants.DbTypeSqlite, constants.DbTypeLibSQL,
		constants.DbTypeMysql, constants.DbTypeMariaDB, constants.DbTypeSqlserver,
		constants.DbTypeYugabyte, constants.DbTypeCockroachDB, constants.DbTypePlanetScaleDB:
		return true
	default:
		return false
	}
}

// testSQLCRUDCorrectnessFixes covers the SQL storage CRUD-correctness fixes:
// #1 AddUser email uniqueness when a phone is also supplied, #2 anchored
// session-token deletion, and #3 the UpdateUser partial-struct guard.
func testSQLCRUDCorrectnessFixes(t *testing.T, ctx context.Context, provider Provider) {
	// #1: AddUser must check email uniqueness even when a phone number is also
	// supplied (previously an else-if skipped the email check).
	sharedEmail := "crud_" + uuid.New().String() + "@test.com"
	userA := &schemas.User{
		ID:            uuid.New().String(),
		Email:         refs.NewStringRef(sharedEmail),
		PhoneNumber:   refs.NewStringRef("phoneA-" + uuid.New().String()),
		Password:      refs.NewStringRef("hashA"),
		SignupMethods: "basic_auth",
	}
	_, err := provider.AddUser(ctx, userA)
	require.NoError(t, err)

	userB := &schemas.User{
		ID:            uuid.New().String(),
		Email:         refs.NewStringRef(sharedEmail), // duplicate email
		PhoneNumber:   refs.NewStringRef("phoneB-" + uuid.New().String()),
		Password:      refs.NewStringRef("hashB"),
		SignupMethods: "basic_auth",
	}
	_, err = provider.AddUser(ctx, userB)
	require.Error(t, err, "duplicate email must be rejected even when phone differs")
	assert.Contains(t, err.Error(), "email", "error should identify the email conflict")

	require.NoError(t, provider.DeleteUser(ctx, userA))

	// #2: DeleteAllSessionTokensByUserID must delete only the target user's
	// tokens, not another user whose stored id contains the target as a
	// substring. otherUser ("xu"+suffix) contains targetUser ("u"+suffix).
	suffix := uuid.New().String()
	targetUser := "u" + suffix
	otherUser := "xu" + suffix
	targetStored := "auth_provider:" + targetUser
	otherStored := "auth_provider:" + otherUser

	addToken := func(storedUserID, key string) {
		require.NoError(t, provider.AddSessionToken(ctx, &schemas.SessionToken{
			UserID:    storedUserID,
			KeyName:   key,
			Token:     "tok_" + uuid.New().String(),
			ExpiresAt: time.Now().Add(60 * time.Second).Unix(),
		}))
	}
	addToken(targetStored, "session_token_key")
	addToken(otherStored, "session_token_key")

	require.NoError(t, provider.DeleteAllSessionTokensByUserID(ctx, targetUser))

	_, err = provider.GetSessionTokenByUserIDAndKey(ctx, targetStored, "session_token_key")
	assert.Error(t, err, "target user's session token should be deleted")
	_, err = provider.GetSessionTokenByUserIDAndKey(ctx, otherStored, "session_token_key")
	assert.NoError(t, err, "substring-matching other user's session token must survive")

	// cleanup the surviving token
	_ = provider.DeleteAllSessionTokensByUserID(ctx, otherUser)

	// #3a: UpdateUser rejects a partial struct (CreatedAt == 0, i.e. never loaded).
	_, err = provider.UpdateUser(ctx, &schemas.User{
		ID:    uuid.New().String(),
		Email: refs.NewStringRef("partial_" + uuid.New().String() + "@test.com"),
	})
	require.Error(t, err, "partial struct with zero CreatedAt must be rejected")
	assert.Contains(t, err.Error(), "created_at")

	// #3b: loading a user, mutating one field and saving must not blank other fields.
	u := &schemas.User{
		ID:            uuid.New().String(),
		Email:         refs.NewStringRef("preserve_" + uuid.New().String() + "@test.com"),
		Password:      refs.NewStringRef("secret-hash"),
		Roles:         "user",
		SignupMethods: "basic_auth",
	}
	created, err := provider.AddUser(ctx, u)
	require.NoError(t, err)

	loaded, err := provider.GetUserByID(ctx, created.ID)
	require.NoError(t, err)
	loaded.GivenName = refs.NewStringRef("UpdatedName")
	_, err = provider.UpdateUser(ctx, loaded)
	require.NoError(t, err)

	refetched, err := provider.GetUserByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "UpdatedName", refs.StringValue(refetched.GivenName))
	assert.Equal(t, "secret-hash", refs.StringValue(refetched.Password), "password must be preserved")
	assert.Equal(t, "user", refetched.Roles, "roles must be preserved")

	require.NoError(t, provider.DeleteUser(ctx, refetched))
}

// testUserSearchOperations verifies ListUsers' optional case-insensitive
// substring filter across email/given_name/family_name/nickname, and that
// pagination + total reflect the filtered set. Every backend must honour it,
// including the O(n)-scan DynamoDB/Cassandra paths.
func testUserSearchOperations(t *testing.T, ctx context.Context, provider Provider) {
	// A unique token shared by the users we create isolates this test from any
	// users already stored by earlier subtests.
	token := "zz" + strings.ReplaceAll(uuid.New().String(), "-", "")
	makeUser := func(email string, given, nick *string) {
		u := &schemas.User{
			ID:            uuid.New().String(),
			Email:         refs.NewStringRef(email),
			Password:      refs.NewStringRef("hashedPassword"),
			SignupMethods: "basic_auth",
			GivenName:     given,
			Nickname:      nick,
		}
		_, err := provider.AddUser(ctx, u)
		require.NoError(t, err)
	}

	// Matches on email.
	makeUser("alice_"+token+"@test.com", nil, nil)
	// Matches on given_name, stored uppercased to prove case-insensitivity.
	makeUser("bob_"+uuid.New().String()+"@test.com", refs.NewStringRef(strings.ToUpper(token)+"Given"), nil)
	// Matches on nickname.
	makeUser("carol_"+uuid.New().String()+"@test.com", nil, refs.NewStringRef("nick"+token))
	// Must NOT match the token.
	makeUser("dave_"+uuid.New().String()+"@test.com", refs.NewStringRef("unrelated"), nil)

	// Case-insensitive search on the shared token returns exactly the 3 matches.
	res, pagination, err := provider.ListUsers(ctx, &model.Pagination{Limit: 50, Offset: 0}, strings.ToUpper(token))
	require.NoError(t, err)
	assert.Equal(t, int64(3), pagination.Total, "search total must count only matching users")
	assert.Len(t, res, 3)

	// Empty query is unfiltered: total must include at least the 4 created here.
	_, allPagination, err := provider.ListUsers(ctx, &model.Pagination{Limit: 1, Offset: 0}, "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, allPagination.Total, int64(4))

	// Pagination applies to the filtered set: limit 2 returns 2 of 3, total 3.
	page1, page1Pagination, err := provider.ListUsers(ctx, &model.Pagination{Limit: 2, Offset: 0}, token)
	require.NoError(t, err)
	assert.Equal(t, int64(3), page1Pagination.Total)
	assert.Len(t, page1, 2)
	page2, _, err := provider.ListUsers(ctx, &model.Pagination{Limit: 2, Offset: 2}, token)
	require.NoError(t, err)
	assert.Len(t, page2, 1)

	// A non-matching query returns nothing.
	none, nonePagination, err := provider.ListUsers(ctx, &model.Pagination{Limit: 50, Offset: 0}, "no-such-user-"+uuid.New().String())
	require.NoError(t, err)
	assert.Equal(t, int64(0), nonePagination.Total)
	assert.Len(t, none, 0)

	// Search also matches on the user id: a prefix of a user's id returns that
	// user even when no other field contains the query. A uuid prefix is unique
	// enough to identify exactly one user.
	idUser := &schemas.User{
		ID:            uuid.New().String(),
		Email:         refs.NewStringRef("erin_" + uuid.New().String() + "@test.com"),
		Password:      refs.NewStringRef("hashedPassword"),
		SignupMethods: "basic_auth",
	}
	_, err = provider.AddUser(ctx, idUser)
	require.NoError(t, err)
	byID, _, err := provider.ListUsers(ctx, &model.Pagination{Limit: 50, Offset: 0}, idUser.ID[:13])
	require.NoError(t, err)
	foundByID := false
	for _, u := range byID {
		if u.ID == idUser.ID {
			foundByID = true
			break
		}
	}
	assert.True(t, foundByID, "search by id prefix must return the user")
}

func testUserOperations(t *testing.T, ctx context.Context, provider Provider, dbType string) {
	// Create test user
	user := &schemas.User{
		ID:            uuid.New().String(),
		Email:         refs.NewStringRef("test_" + uuid.New().String() + "@test.com"),
		Password:      refs.NewStringRef("hashedPassword"),
		SignupMethods: "basic_auth",
	}

	// Test AddUser
	createdUser, err := provider.AddUser(ctx, user)
	require.NoError(t, err)
	require.NotNil(t, createdUser)
	assert.Equal(t, user.Email, createdUser.Email)

	// Test GetUserByEmail
	fetchedUser, err := provider.GetUserByEmail(ctx, *user.Email)
	require.NoError(t, err)
	require.NotNil(t, fetchedUser)
	assert.Equal(t, user.Email, fetchedUser.Email)

	// Password is tagged json:"-" for API safety but must still persist to and load
	// from the database. Couchbase (de)serializes structs via encoding/json, which
	// honors json:"-" and previously dropped the password entirely.
	// Regression guard: fix/couchbase-secret-json-tag-persistence.
	require.NotNil(t, fetchedUser.Password, "stored password must round-trip from the database")
	assert.Equal(t, *user.Password, *fetchedUser.Password)

	// Test UpdateUser mutates only the changed field and preserves the rest.
	fetchedUser.GivenName = refs.NewStringRef("Updated")
	updatedUser, err := provider.UpdateUser(ctx, fetchedUser)
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, "Updated", *updatedUser.GivenName)
	// Reload and assert unrelated fields were not blanked by the full-document write.
	reloadedUser, err := provider.GetUserByID(ctx, createdUser.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", *reloadedUser.GivenName)
	assert.Equal(t, *user.Email, *reloadedUser.Email, "email must survive an unrelated update")
	assert.Equal(t, "basic_auth", reloadedUser.SignupMethods, "signup_methods must survive an unrelated update")
	require.NotNil(t, reloadedUser.Password, "password must survive an unrelated update")
	assert.Equal(t, *user.Password, *reloadedUser.Password)

	if dbType == constants.DbTypeMongoDB {
		// Guard: a partial struct with no CreatedAt (caller forgot to load the
		// record first) must be rejected, not silently blank every other field.
		partial := &schemas.User{
			ID:        createdUser.ID,
			GivenName: refs.NewStringRef("ShouldNotPersist"),
		}
		_, err = provider.UpdateUser(ctx, partial)
		require.Error(t, err, "UpdateUser must reject a partial struct with zero CreatedAt")
		assert.Contains(t, err.Error(), "partial struct detected")
		// The stored document must be untouched by the rejected write.
		intact, err := provider.GetUserByID(ctx, createdUser.ID)
		require.NoError(t, err)
		require.NotNil(t, intact.Email, "rejected update must not blank the email")
		assert.Equal(t, *user.Email, *intact.Email)
		assert.Equal(t, "Updated", *intact.GivenName, "rejected update must not overwrite given_name")
	}

	// Nullable pointer fields must clear to null when set to nil on update.
	// Regression guard: a nil pointer is omitted from the update, so a provider
	// that does not explicitly clear the stored attribute (DynamoDB) would leave
	// the stale value behind (e.g. an avatar URL that can never be removed).
	fetchedUser.Picture = refs.NewStringRef("https://example.com/avatar.png")
	_, err = provider.UpdateUser(ctx, fetchedUser)
	require.NoError(t, err)
	withPicture, err := provider.GetUserByID(ctx, fetchedUser.ID)
	require.NoError(t, err)
	require.NotNil(t, withPicture.Picture)
	assert.Equal(t, "https://example.com/avatar.png", *withPicture.Picture)

	withPicture.Picture = nil
	_, err = provider.UpdateUser(ctx, withPicture)
	require.NoError(t, err)
	cleared, err := provider.GetUserByID(ctx, fetchedUser.ID)
	require.NoError(t, err)
	assert.Nil(t, cleared.Picture, "Picture must be cleared to nil, not left at the stale stored value")

	// Test ListUsers
	users, pagination, err := provider.ListUsers(ctx, &model.Pagination{
		Limit:  10,
		Offset: 0,
	}, "")
	assert.NoError(t, err)
	assert.NotNil(t, users)
	assert.Greater(t, len(users), 0)
	assert.NotNil(t, pagination)
	assert.Greater(t, pagination.Total, int64(0))

	// Test DeleteUser
	err = provider.DeleteUser(ctx, user)
	assert.NoError(t, err)

	// Verify deletion
	_, err = provider.GetUserByEmail(ctx, *user.Email)
	assert.Error(t, err)

	// Test GetUserByPhoneNumber
	user.PhoneNumber = refs.NewStringRef("+1234567891")
	createdUser, err = provider.AddUser(ctx, user)
	require.NoError(t, err)
	require.NotNil(t, createdUser)
	assert.Equal(t, user.PhoneNumber, createdUser.PhoneNumber)

	// Test GetUserByID
	fetchedUser, err = provider.GetUserByID(ctx, createdUser.ID)
	require.NoError(t, err)
	require.NotNil(t, fetchedUser)
	assert.Equal(t, user.PhoneNumber, fetchedUser.PhoneNumber)
	// Password must round-trip through the GetUserByID read path too, not just
	// GetUserByEmail — each read method is a distinct provider call site.
	require.NotNil(t, fetchedUser.Password, "GetUserByID must round-trip the stored password")
	assert.Equal(t, *user.Password, *fetchedUser.Password)

	// Test UpdateUsers
	users, _, err = provider.ListUsers(ctx, &model.Pagination{
		Limit:  10,
		Offset: 0,
	}, "")
	assert.NoError(t, err)
	assert.NotNil(t, users)
	assert.Greater(t, len(users), 0)
	// ListUsers is a third read path that could silently drop the json:"-"
	// Password field; verify the created user carries it in the list result.
	var foundInList *schemas.User
	for _, u := range users {
		if u.ID == createdUser.ID {
			foundInList = u
			break
		}
	}
	require.NotNil(t, foundInList, "created user should appear in ListUsers")
	require.NotNil(t, foundInList.Password, "ListUsers must not drop the stored password")
	assert.Equal(t, *user.Password, *foundInList.Password)
	data := map[string]interface{}{
		"phone_number": "+3216549870",
	}
	err = provider.UpdateUsers(ctx, data, []string{createdUser.ID})
	assert.NoError(t, err)

	// Test GetUserByPhoneNumber after update
	user, err = provider.GetUserByPhoneNumber(ctx, "+3216549870")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "+3216549870", *user.PhoneNumber)

	// Cleanup: delete the user to avoid data leakage between test runs
	err = provider.DeleteUser(ctx, user)
	assert.NoError(t, err)

}

func testVerificationRequestOperations(t *testing.T, ctx context.Context, provider Provider, dbType string) {
	vr := &schemas.VerificationRequest{
		Token:       uuid.New().String(),
		Email:       "test_" + uuid.New().String() + "@test.com",
		ExpiresAt:   time.Now().Add(24 * time.Hour).Unix(),
		Identifier:  "email_verification",
		Nonce:       uuid.New().String(),
		RedirectURI: "https://app.example.com/callback",
	}

	// Test AddVerificationRequest
	created, err := provider.AddVerificationRequest(ctx, vr)
	assert.NoError(t, err)
	assert.NotNil(t, created)

	// Test GetVerificationRequestByToken
	fetched, err := provider.GetVerificationRequestByToken(ctx, vr.Token)
	assert.NoError(t, err)
	assert.Equal(t, vr.Email, fetched.Email)

	// Test GetVerificationRequestByEmail
	fetchedByEmail, err := provider.GetVerificationRequestByEmail(ctx, vr.Email, vr.Identifier)
	assert.NoError(t, err)
	assert.Equal(t, vr.Token, fetchedByEmail.Token)

	// Test ListVerificationRequests
	requests, _, err := provider.ListVerificationRequests(ctx, &model.Pagination{
		Limit:  10,
		Offset: 0,
	})
	assert.NoError(t, err)
	assert.NotNil(t, requests)
	assert.Greater(t, len(requests), 0)

	// ListVerificationRequests must return the real record fields, not empty strings.
	// (Regression guard: the Couchbase provider previously SELECTed a non-existent `env`
	// column, so every listed record had empty token/identifier/email/nonce/redirect_uri.)
	var listed *schemas.VerificationRequest
	for _, r := range requests {
		if r.Token == vr.Token {
			listed = r
			break
		}
	}
	require.NotNil(t, listed, "created verification request must appear in ListVerificationRequests")
	assert.Equal(t, vr.Email, listed.Email)
	assert.Equal(t, vr.Identifier, listed.Identifier)
	assert.Equal(t, vr.Nonce, listed.Nonce)
	assert.Equal(t, vr.RedirectURI, listed.RedirectURI)
	assert.Equal(t, vr.ExpiresAt, listed.ExpiresAt)

	// A second AddVerificationRequest for the same (email, identifier) - the
	// normal resend-verification-email flow always deletes the old request
	// first (see resend_verify_email.go), so this exercises what happens if
	// that invariant is ever violated (e.g. a caller bug, or a race between
	// two concurrent resends). Backend behavior intentionally differs here:
	// SQL upserts on conflict (clause.OnConflict...DoUpdates, so the second
	// call succeeds and replaces the pending request in place); Mongo has no
	// such upsert logic, so its unique index must hard-reject instead, or a
	// second silent row would accumulate.
	dupToken := uuid.New().String()
	dup, dupErr := provider.AddVerificationRequest(ctx, &schemas.VerificationRequest{
		Token:       dupToken,
		Email:       vr.Email,
		ExpiresAt:   time.Now().Add(24 * time.Hour).Unix(),
		Identifier:  vr.Identifier,
		Nonce:       uuid.New().String(),
		RedirectURI: "https://app.example.com/callback",
	})
	if isSQLTestDB(dbType) {
		assert.NoError(t, dupErr, "SQL upserts a duplicate (email, identifier) request rather than erroring")
		refetched, err := provider.GetVerificationRequestByEmail(ctx, vr.Email, vr.Identifier)
		require.NoError(t, err)
		assert.Equal(t, dupToken, refetched.Token, "upsert must replace the pending request, not add a second one")
	} else if dbType == constants.DbTypeMongoDB {
		// Regression guard: this compound index used a multi-key bson.M, which
		// the driver silently rejects at CreateIndexes time - the constraint
		// was never actually created, so this duplicate would have been
		// accepted, leaving two pending requests for the same identity.
		assert.Error(t, dupErr, "duplicate (email, identifier) verification request must be rejected")
	} else if dupErr == nil {
		// Cassandra/DynamoDB/ArangoDB/Couchbase: neither upsert nor a unique
		// constraint exist yet for this pair - known gap, not fixed here.
		// Clean up the extra row so it doesn't leak into other subtests.
		_ = provider.DeleteVerificationRequest(ctx, dup)
	}

	// Test DeleteVerificationRequest
	err = provider.DeleteVerificationRequest(ctx, vr)
	assert.NoError(t, err)
}

func testSessionOperations(t *testing.T, ctx context.Context, provider Provider) {
	userID := uuid.New().String()
	session := &schemas.Session{
		UserID:    userID,
		UserAgent: "test_user_agent",
		IP:        "127.0.0.1",
	}

	// Test AddSession
	err := provider.AddSession(ctx, session)
	assert.NoError(t, err)

	// Test DeleteSession
	err = provider.DeleteSession(ctx, userID)
	assert.NoError(t, err)
}

func testWebhookOperations(t *testing.T, ctx context.Context, provider Provider, dbType string) {
	webhook := &schemas.Webhook{
		EventName: "test_event",
		EndPoint:  "https://test.com/webhook",
		Enabled:   true,
	}

	// Test AddWebhook
	created, err := provider.AddWebhook(ctx, webhook)
	assert.NoError(t, err)
	assert.NotNil(t, created)

	// Test GetWebhookByID
	fetched, err := provider.GetWebhookByID(ctx, created.ID)
	assert.NoError(t, err)
	assert.Equal(t, webhook.EventName, fetched.EventName)

	// Test GetWebhookByEventName
	fetchedByEventName, err := provider.GetWebhookByEventName(ctx, webhook.EventName)
	assert.NoError(t, err)
	assert.NotNil(t, fetchedByEventName)
	assert.Equal(t, created.ID, fetchedByEventName[0].ID)

	// Test UpdateWebhook mutates only the changed field and preserves the rest.
	webhook.EndPoint = "https://test.com/webhook_updated"
	updated, err := provider.UpdateWebhook(ctx, webhook)
	assert.NoError(t, err)
	assert.Equal(t, webhook.EndPoint, updated.EndPoint)
	// Reload and assert unrelated fields were not blanked by the full-document write.
	reloadedWebhook, err := provider.GetWebhookByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, updated.EndPoint, reloadedWebhook.EndPoint)
	assert.Equal(t, created.EventName, reloadedWebhook.EventName, "event_name must survive an unrelated update")
	assert.Equal(t, created.Enabled, reloadedWebhook.Enabled, "enabled must survive an unrelated update")

	if dbType == constants.DbTypeMongoDB {
		// Guard: a partial struct with no CreatedAt must be rejected.
		partial := &schemas.Webhook{
			ID:       created.ID,
			EndPoint: "https://should-not-persist.example.com",
		}
		_, err = provider.UpdateWebhook(ctx, partial)
		require.Error(t, err, "UpdateWebhook must reject a partial struct with zero CreatedAt")
		assert.Contains(t, err.Error(), "partial struct detected")
		intact, err := provider.GetWebhookByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.EventName, intact.EventName, "rejected update must not blank event_name")
		assert.Equal(t, updated.EndPoint, intact.EndPoint, "rejected update must not overwrite endpoint")
	}

	// Test ListWebhook
	webhooks, _, err := provider.ListWebhook(ctx, &model.Pagination{
		Limit:  10,
		Offset: 0,
	})
	assert.NoError(t, err)
	assert.NotNil(t, webhooks)
	assert.Greater(t, len(webhooks), 0)

	// Test ListWebhookLogs
	logs, _, err := provider.ListWebhookLogs(ctx, &model.Pagination{
		Limit:  10,
		Offset: 0,
	}, webhook.ID)
	assert.NoError(t, err)
	assert.NotNil(t, logs)
	assert.Empty(t, len(logs))

	// Test DeleteWebhook
	err = provider.DeleteWebhook(ctx, updated)
	assert.NoError(t, err)
}

func testEmailTemplateOperations(t *testing.T, ctx context.Context, provider Provider, dbType string) {
	template := &schemas.EmailTemplate{
		EventName: "test_event_" + uuid.New().String(),
		Template:  "Test template",
		Subject:   "Test subject",
	}

	// Test AddEmailTemplate
	created, err := provider.AddEmailTemplate(ctx, template)
	assert.NoError(t, err)
	assert.NotNil(t, created)

	// Test GetEmailTemplateByID
	fetched, err := provider.GetEmailTemplateByID(ctx, template.ID)
	assert.NoError(t, err)
	assert.Equal(t, template.EventName, fetched.EventName)

	// Test GetEmailTemplateByEventName
	fetchedByEventName, err := provider.GetEmailTemplateByEventName(ctx, template.EventName)
	assert.NoError(t, err)
	assert.NotNil(t, fetchedByEventName)
	assert.Equal(t, template.EventName, fetchedByEventName.EventName)
	assert.Equal(t, created.ID, fetchedByEventName.ID)

	// Test UpdateEmailTemplate mutates only the changed field and preserves the rest.
	template.Template = "Updated template"
	updated, err := provider.UpdateEmailTemplate(ctx, template)
	assert.NoError(t, err)
	assert.Equal(t, template.Template, updated.Template)
	// Reload and assert unrelated fields were not blanked by the full-document write.
	reloadedTemplate, err := provider.GetEmailTemplateByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated template", reloadedTemplate.Template)
	assert.Equal(t, created.EventName, reloadedTemplate.EventName, "event_name must survive an unrelated update")
	assert.Equal(t, created.Subject, reloadedTemplate.Subject, "subject must survive an unrelated update")

	if dbType == constants.DbTypeMongoDB {
		// Guard: a partial struct with no CreatedAt must be rejected.
		partial := &schemas.EmailTemplate{
			ID:       created.ID,
			Template: "should not persist",
		}
		_, err = provider.UpdateEmailTemplate(ctx, partial)
		require.Error(t, err, "UpdateEmailTemplate must reject a partial struct with zero CreatedAt")
		assert.Contains(t, err.Error(), "partial struct detected")
		intact, err := provider.GetEmailTemplateByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.EventName, intact.EventName, "rejected update must not blank event_name")
		assert.Equal(t, created.Subject, intact.Subject, "rejected update must not blank subject")
		assert.Equal(t, "Updated template", intact.Template, "rejected update must not overwrite template")
	}

	// Test ListEmailTemplate
	templates, _, err := provider.ListEmailTemplate(ctx, &model.Pagination{
		Limit:  10,
		Offset: 0,
	})
	assert.NoError(t, err)
	assert.NotNil(t, templates)
	assert.Greater(t, len(templates), 0)
	found := false
	for _, tmpl := range templates {
		if tmpl.EventName == template.EventName {
			found = true
			assert.Equal(t, template.Template, tmpl.Template)
			break
		}
	}
	assert.True(t, found, "expected updated template in listed templates")

	// Test DeleteEmailTemplate
	err = provider.DeleteEmailTemplate(ctx, updated)
	assert.NoError(t, err)

	// Test GetEmailTemplateByEventName after delete
	fetchedByEventName, err = provider.GetEmailTemplateByEventName(ctx, template.EventName)
	assert.Error(t, err)
	assert.Nil(t, fetchedByEventName)
}

func testOTPOperations(t *testing.T, ctx context.Context, provider Provider) {
	otp := &schemas.OTP{
		Email:     "test_" + uuid.New().String() + "@test.com",
		Otp:       "123456",
		ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
	}

	// Test UpsertOTP
	created, err := provider.UpsertOTP(ctx, otp)
	assert.NoError(t, err)
	assert.NotNil(t, created)

	// Test GetOTPByEmail
	fetched, err := provider.GetOTPByEmail(ctx, otp.Email)
	assert.NoError(t, err)
	assert.Equal(t, otp.Otp, fetched.Otp)

	// For same email address, upsert should update the OTP
	otp.Otp = "789012"
	updated, err := provider.UpsertOTP(ctx, otp)
	assert.NoError(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, otp.Otp, updated.Otp)

	// Test DeleteOTP
	err = provider.DeleteOTP(ctx, updated)
	assert.NoError(t, err)
}

func testAuthenticatorOperations(t *testing.T, ctx context.Context, provider Provider, dbType string) {
	auth := &schemas.Authenticator{
		UserID:        uuid.New().String(),
		Method:        constants.EnvKeyTOTPAuthenticator,
		RecoveryCodes: refs.NewStringRef("test"),
		Secret:        "test_secret",
	}

	// Test AddAuthenticator
	created, err := provider.AddAuthenticator(ctx, auth)
	assert.NoError(t, err)
	assert.NotNil(t, created)

	// Test GetAuthenticatorDetailsByUserId
	fetched, err := provider.GetAuthenticatorDetailsByUserId(ctx, auth.UserID, constants.EnvKeyTOTPAuthenticator)
	assert.NoError(t, err)
	assert.Equal(t, auth.Secret, fetched.Secret)

	// Test UpdateAuthenticator mutates only the changed field and preserves the rest.
	auth.Secret = "updated_secret"
	updated, err := provider.UpdateAuthenticator(ctx, auth)
	assert.NoError(t, err)
	assert.Equal(t, "updated_secret", updated.Secret)
	// Reload and assert unrelated fields were not blanked by the full-document write.
	reloaded, err := provider.GetAuthenticatorDetailsByUserId(ctx, auth.UserID, constants.EnvKeyTOTPAuthenticator)
	require.NoError(t, err)
	assert.Equal(t, "updated_secret", reloaded.Secret)
	assert.Equal(t, auth.Method, reloaded.Method, "method must survive an unrelated update")
	require.NotNil(t, reloaded.RecoveryCodes, "recovery_codes must survive an unrelated update")
	assert.Equal(t, "test", *reloaded.RecoveryCodes)

	if dbType == constants.DbTypeMongoDB {
		// Guard: a partial struct with no CreatedAt must be rejected, not silently
		// blank the recovery codes / secret of an enrolled authenticator.
		partial := &schemas.Authenticator{
			ID:     created.ID,
			Secret: "should_not_persist",
		}
		_, err = provider.UpdateAuthenticator(ctx, partial)
		require.Error(t, err, "UpdateAuthenticator must reject a partial struct with zero CreatedAt")
		assert.Contains(t, err.Error(), "partial struct detected")
		intact, err := provider.GetAuthenticatorDetailsByUserId(ctx, auth.UserID, constants.EnvKeyTOTPAuthenticator)
		require.NoError(t, err)
		assert.Equal(t, "updated_secret", intact.Secret, "rejected update must not overwrite secret")
		require.NotNil(t, intact.RecoveryCodes, "rejected update must not blank recovery_codes")

		// Enrollment race: a second enrollment for the same (user_id, method) must
		// not create a divergent duplicate. AddAuthenticator's pre-check returns the
		// existing record; the unique (user_id, method) index backstops a true race.
		dup := &schemas.Authenticator{
			UserID: auth.UserID,
			Method: constants.EnvKeyTOTPAuthenticator,
			Secret: "second_enrollment_secret",
		}
		_, err = provider.AddAuthenticator(ctx, dup)
		require.NoError(t, err)
		afterDup, err := provider.GetAuthenticatorDetailsByUserId(ctx, auth.UserID, constants.EnvKeyTOTPAuthenticator)
		require.NoError(t, err)
		assert.Equal(t, "updated_secret", afterDup.Secret, "second enrollment must not create a divergent duplicate")
	}
}

func testSessionTokenOperations(t *testing.T, ctx context.Context, provider Provider) {
	userId := "auth_provider:" + uuid.New().String()
	token1 := &schemas.SessionToken{
		UserID:    userId,
		KeyName:   "session_token_key",
		Token:     "test_hash_token",
		ExpiresAt: time.Now().Add(60 * time.Second).Unix(),
	}

	// Test AddSessionToken
	err := provider.AddSessionToken(ctx, token1)
	require.NoError(t, err)
	require.NotEmpty(t, token1.ID)

	// Test GetSessionTokenByUserIDAndKey
	fetched, err := provider.GetSessionTokenByUserIDAndKey(ctx, userId, "session_token_key")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, token1.Token, fetched.Token)
	assert.Equal(t, userId, fetched.UserID)

	// Test AddSessionToken with same key (should replace - delete first then add)
	// Note: Multiple sessions per user are allowed (different key_name), but same (user_id, key_name) should be unique
	err = provider.DeleteSessionTokenByUserIDAndKey(ctx, userId, "session_token_key")
	assert.NoError(t, err)

	token2 := &schemas.SessionToken{
		UserID:    userId,
		KeyName:   "session_token_key",
		Token:     "updated_hash_token",
		ExpiresAt: time.Now().Add(60 * time.Second).Unix(),
	}
	err = provider.AddSessionToken(ctx, token2)
	require.NoError(t, err)

	// Verify it was updated
	fetched, err = provider.GetSessionTokenByUserIDAndKey(ctx, userId, "session_token_key")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "updated_hash_token", fetched.Token)

	// Test AddSessionToken with different key
	token3 := &schemas.SessionToken{
		UserID:    userId,
		KeyName:   "access_token_key",
		Token:     "test_access_token",
		ExpiresAt: time.Now().Add(60 * time.Second).Unix(),
	}
	err = provider.AddSessionToken(ctx, token3)
	require.NoError(t, err)

	// Test DeleteSessionTokenByUserIDAndKey
	err = provider.DeleteSessionTokenByUserIDAndKey(ctx, userId, "session_token_key")
	assert.NoError(t, err)

	// Verify deletion
	_, err = provider.GetSessionTokenByUserIDAndKey(ctx, userId, "session_token_key")
	assert.Error(t, err)

	// Verify other key still exists
	fetched, err = provider.GetSessionTokenByUserIDAndKey(ctx, userId, "access_token_key")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, token3.Token, fetched.Token)

	// Test DeleteSessionToken by ID
	err = provider.DeleteSessionToken(ctx, fetched.ID)
	require.NoError(t, err)

	// Test DeleteAllSessionTokensByUserID
	userId2 := "auth_provider:" + uuid.New().String()
	token4 := &schemas.SessionToken{
		UserID:    userId2,
		KeyName:   "session_token_key",
		Token:     "test_token_4",
		ExpiresAt: time.Now().Add(60 * time.Second).Unix(),
	}
	token5 := &schemas.SessionToken{
		UserID:    userId2,
		KeyName:   "access_token_key",
		Token:     "test_token_5",
		ExpiresAt: time.Now().Add(60 * time.Second).Unix(),
	}
	err = provider.AddSessionToken(ctx, token4)
	require.NoError(t, err)
	err = provider.AddSessionToken(ctx, token5)
	require.NoError(t, err)

	// A different user's token must NOT be swept up by DeleteAllSessionTokensByUserID.
	// user_id is stored namespaced ("auth_provider:<uuid>") and the delete is called with
	// the bare uuid, so the match is a suffix/substring — a second, distinct user's token
	// (different uuid) must survive to prove no cross-user collateral deletion.
	otherUser := "auth_provider:" + uuid.New().String()
	otherToken := &schemas.SessionToken{
		UserID:    otherUser,
		KeyName:   "session_token_key",
		Token:     "other_user_token",
		ExpiresAt: time.Now().Add(60 * time.Second).Unix(),
	}
	err = provider.AddSessionToken(ctx, otherToken)
	require.NoError(t, err)

	// Extract just the user ID part for DeleteAllSessionTokensByUserID
	userIDPart := userId2[len("auth_provider:"):]
	err = provider.DeleteAllSessionTokensByUserID(ctx, userIDPart)
	assert.NoError(t, err)

	// Verify all sessions for user are deleted
	_, err = provider.GetSessionTokenByUserIDAndKey(ctx, userId2, "session_token_key")
	assert.Error(t, err)
	_, err = provider.GetSessionTokenByUserIDAndKey(ctx, userId2, "access_token_key")
	assert.Error(t, err)

	// The other user's session must still exist.
	otherFetched, err := provider.GetSessionTokenByUserIDAndKey(ctx, otherUser, "session_token_key")
	require.NoError(t, err, "deleting one user's sessions must not delete a distinct user's sessions")
	require.NotNil(t, otherFetched)
	assert.Equal(t, "other_user_token", otherFetched.Token)

	// Clean up the other user's token so it does not leak into later assertions.
	err = provider.DeleteAllSessionTokensByUserID(ctx, otherUser[len("auth_provider:"):])
	assert.NoError(t, err)

	// Test DeleteSessionTokensByNamespace
	namespace := "auth_provider"
	token6 := &schemas.SessionToken{
		UserID:    namespace + ":user1",
		KeyName:   "session_token_key",
		Token:     "test_token_6",
		ExpiresAt: time.Now().Add(60 * time.Second).Unix(),
	}
	err = provider.AddSessionToken(ctx, token6)
	require.NoError(t, err)

	err = provider.DeleteSessionTokensByNamespace(ctx, namespace)
	assert.NoError(t, err)

	// Verify namespace sessions are deleted
	_, err = provider.GetSessionTokenByUserIDAndKey(ctx, namespace+":user1", "session_token_key")
	assert.Error(t, err)

	// Test CleanExpiredSessionTokens
	expiredToken := &schemas.SessionToken{
		UserID:    "auth_provider:expired_user",
		KeyName:   "session_token_key",
		Token:     "expired_token",
		ExpiresAt: time.Now().Add(-60 * time.Second).Unix(), // Already expired
	}
	err = provider.AddSessionToken(ctx, expiredToken)
	require.NoError(t, err)

	err = provider.CleanExpiredSessionTokens(ctx)
	assert.NoError(t, err)

	// Verify expired token is cleaned
	_, err = provider.GetSessionTokenByUserIDAndKey(ctx, "auth_provider:expired_user", "session_token_key")
	assert.Error(t, err)
}

func testMFASessionOperations(t *testing.T, ctx context.Context, provider Provider) {
	userId := "auth_provider:" + uuid.New().String()
	mfaSession1 := &schemas.MFASession{
		UserID:    userId,
		KeyName:   "mfa_session_key_1",
		ExpiresAt: time.Now().Add(60 * time.Second).Unix(),
	}

	// Test AddMFASession
	err := provider.AddMFASession(ctx, mfaSession1)
	require.NoError(t, err)
	require.NotEmpty(t, mfaSession1.ID)

	// Test GetMFASessionByUserIDAndKey
	fetched, err := provider.GetMFASessionByUserIDAndKey(ctx, userId, "mfa_session_key_1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, userId, fetched.UserID)
	assert.Equal(t, "mfa_session_key_1", fetched.KeyName)

	// Test AddMFASession with same key (should replace - delete first then add)
	// Note: Multiple MFA sessions per user are allowed (different key_name), but same (user_id, key_name) should be unique
	err = provider.DeleteMFASessionByUserIDAndKey(ctx, userId, "mfa_session_key_1")
	assert.NoError(t, err)

	mfaSession2 := &schemas.MFASession{
		UserID:    userId,
		KeyName:   "mfa_session_key_1",
		ExpiresAt: time.Now().Add(120 * time.Second).Unix(),
	}
	err = provider.AddMFASession(ctx, mfaSession2)
	require.NoError(t, err)

	// Verify it was updated
	fetched, err = provider.GetMFASessionByUserIDAndKey(ctx, userId, "mfa_session_key_1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, mfaSession2.ExpiresAt, fetched.ExpiresAt)

	// Test AddMFASession with different key
	mfaSession3 := &schemas.MFASession{
		UserID:    userId,
		KeyName:   "mfa_session_key_2",
		ExpiresAt: time.Now().Add(60 * time.Second).Unix(),
	}
	err = provider.AddMFASession(ctx, mfaSession3)
	require.NoError(t, err)

	// Test GetAllMFASessionsByUserID
	allSessions, err := provider.GetAllMFASessionsByUserID(ctx, userId)
	require.NoError(t, err)
	require.NotNil(t, allSessions)
	assert.GreaterOrEqual(t, len(allSessions), 2)

	// Verify both sessions are present
	foundKey1 := false
	foundKey2 := false
	for _, session := range allSessions {
		if session.KeyName == "mfa_session_key_1" {
			foundKey1 = true
		}
		if session.KeyName == "mfa_session_key_2" {
			foundKey2 = true
		}
	}
	assert.True(t, foundKey1, "Should find mfa_session_key_1")
	assert.True(t, foundKey2, "Should find mfa_session_key_2")

	// Test DeleteMFASessionByUserIDAndKey
	err = provider.DeleteMFASessionByUserIDAndKey(ctx, userId, "mfa_session_key_1")
	assert.NoError(t, err)

	// Verify deletion
	_, err = provider.GetMFASessionByUserIDAndKey(ctx, userId, "mfa_session_key_1")
	assert.Error(t, err)

	// Verify other key still exists
	fetched, err = provider.GetMFASessionByUserIDAndKey(ctx, userId, "mfa_session_key_2")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, mfaSession3.UserID, fetched.UserID)

	// Test DeleteMFASession by ID
	err = provider.DeleteMFASession(ctx, fetched.ID)
	require.NoError(t, err)

	// Test CleanExpiredMFASessions
	expiredMFA := &schemas.MFASession{
		UserID:    "auth_provider:expired_user",
		KeyName:   "expired_session_key",
		ExpiresAt: time.Now().Add(-60 * time.Second).Unix(), // Already expired
	}
	err = provider.AddMFASession(ctx, expiredMFA)
	require.NoError(t, err)

	err = provider.CleanExpiredMFASessions(ctx)
	assert.NoError(t, err)

	// Verify expired session is cleaned
	_, err = provider.GetMFASessionByUserIDAndKey(ctx, "auth_provider:expired_user", "expired_session_key")
	assert.Error(t, err)
}

func testAuditLogOperations(t *testing.T, ctx context.Context, provider Provider) {
	t.Run("add and list", func(t *testing.T) {
		auditLog := &schemas.AuditLog{
			ActorID:      uuid.New().String(),
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   "test_" + uuid.New().String() + "@example.com",
			Action:       constants.AuditLoginSuccessEvent,
			ResourceType: constants.AuditResourceTypeSession,
			ResourceID:   uuid.New().String(),
			IPAddress:    "127.0.0.1",
			UserAgent:    "provider-test-agent",
		}
		err := provider.AddAuditLog(ctx, auditLog)
		require.NoError(t, err)
		assert.NotEmpty(t, auditLog.ID)
		assert.NotZero(t, auditLog.CreatedAt)

		pagination := &model.Pagination{Limit: 10, Offset: 0}
		logs, pag, err := provider.ListAuditLogs(ctx, pagination, map[string]interface{}{})
		require.NoError(t, err)
		require.NotNil(t, pag)
		assert.GreaterOrEqual(t, len(logs), 1)
	})

	t.Run("filter by action", func(t *testing.T) {
		uniqueAction := "provider_test_action_" + uuid.New().String()[:8]
		auditLog := &schemas.AuditLog{
			ActorID:      uuid.New().String(),
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   "filter_" + uuid.New().String() + "@example.com",
			Action:       uniqueAction,
			ResourceType: constants.AuditResourceTypeUser,
		}
		require.NoError(t, provider.AddAuditLog(ctx, auditLog))

		pagination := &model.Pagination{Limit: 10, Offset: 0}
		logs, _, err := provider.ListAuditLogs(ctx, pagination, map[string]interface{}{
			"action": uniqueAction,
		})
		require.NoError(t, err)
		require.Len(t, logs, 1)
		assert.Equal(t, uniqueAction, logs[0].Action)
	})

	t.Run("filter by actor_id", func(t *testing.T) {
		actorID := uuid.New().String()
		auditLog := &schemas.AuditLog{
			ActorID:      actorID,
			ActorType:    constants.AuditActorTypeAdmin,
			ActorEmail:   "admin_" + uuid.New().String() + "@example.com",
			Action:       constants.AuditAdminUserUpdatedEvent,
			ResourceType: constants.AuditResourceTypeUser,
		}
		require.NoError(t, provider.AddAuditLog(ctx, auditLog))

		pagination := &model.Pagination{Limit: 10, Offset: 0}
		logs, _, err := provider.ListAuditLogs(ctx, pagination, map[string]interface{}{
			"actor_id": actorID,
		})
		require.NoError(t, err)
		require.Len(t, logs, 1)
		assert.Equal(t, actorID, logs[0].ActorID)
	})

	t.Run("list does not mutate caller pagination pointer", func(t *testing.T) {
		pagination := &model.Pagination{Limit: 10, Offset: 0}
		_, returnedPag, err := provider.ListAuditLogs(ctx, pagination, map[string]interface{}{})
		require.NoError(t, err)
		assert.NotSame(t, pagination, returnedPag, "should return a new pagination object")
	})

	t.Run("delete before created_at", func(t *testing.T) {
		uniqueAction := "provider_cleanup_" + uuid.New().String()[:8]
		oldLog := &schemas.AuditLog{
			ActorID:      uuid.New().String(),
			ActorType:    constants.AuditActorTypeUser,
			ActorEmail:   "system_" + uuid.New().String() + "@example.com",
			Action:       uniqueAction,
			ResourceType: constants.AuditResourceTypeUser,
			CreatedAt:    time.Now().Add(-24 * time.Hour).Unix(),
		}
		require.NoError(t, provider.AddAuditLog(ctx, oldLog))

		before := time.Now().Add(-1 * time.Hour).Unix()
		require.NoError(t, provider.DeleteAuditLogsBefore(ctx, before))

		pagination := &model.Pagination{Limit: 10, Offset: 0}
		logs, _, err := provider.ListAuditLogs(ctx, pagination, map[string]interface{}{
			"action": uniqueAction,
		})
		require.NoError(t, err)
		assert.Empty(t, logs)
	})
}

func testOAuthStateOperations(t *testing.T, ctx context.Context, provider Provider) {
	state1 := &schemas.OAuthState{
		StateKey: "test_state_key_1",
		State:    "test_state_value_1",
	}

	// Test AddOAuthState
	err := provider.AddOAuthState(ctx, state1)
	require.NoError(t, err)
	require.NotEmpty(t, state1.ID)

	// Test GetOAuthStateByKey
	fetched, err := provider.GetOAuthStateByKey(ctx, "test_state_key_1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "test_state_value_1", fetched.State)
	assert.Equal(t, "test_state_key_1", fetched.StateKey)

	// Test AddOAuthState with same key (should replace)
	state2 := &schemas.OAuthState{
		StateKey: "test_state_key_1",
		State:    "updated_state_value",
	}
	err = provider.AddOAuthState(ctx, state2)
	require.NoError(t, err)

	// Verify it was updated
	fetched, err = provider.GetOAuthStateByKey(ctx, "test_state_key_1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "updated_state_value", fetched.State)

	// Test AddOAuthState with different key
	state3 := &schemas.OAuthState{
		StateKey: "test_state_key_2",
		State:    "test_state_value_2",
	}
	err = provider.AddOAuthState(ctx, state3)
	require.NoError(t, err)

	// Test DeleteOAuthStateByKey
	err = provider.DeleteOAuthStateByKey(ctx, "test_state_key_1")
	assert.NoError(t, err)

	// Verify deletion
	_, err = provider.GetOAuthStateByKey(ctx, "test_state_key_1")
	assert.Error(t, err)

	// Verify other key still exists
	fetched, err = provider.GetOAuthStateByKey(ctx, "test_state_key_2")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "test_state_value_2", fetched.State)

	// Test GetAllOAuthStates (for testing purposes)
	allStates, err := provider.GetAllOAuthStates(ctx)
	require.NoError(t, err)
	require.NotNil(t, allStates)
	// Should have at least state2
	found := false
	for _, state := range allStates {
		if state.StateKey == "test_state_key_2" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find test_state_key_2 in all states")
}

// testClientOperations exercises Client CRUD using the
// *externally exposed* id (AsAPIClient().ID) for every lookup — not
// the raw internal ID a provider returns from Add. This distinction matters:
// on ArangoDB the raw internal ID is a "collection/key" handle, while the
// API-facing id (what a real caller receives as client_id, and the only form
// the admin API / token endpoint ever has) is the bare key. A provider whose
// GetClientByID only matches the raw handle will pass a test that
// round-trips through created.ID directly, while silently failing every real
// caller — client_credentials authentication would be completely broken.
func testClientOperations(t *testing.T, ctx context.Context, provider Provider) {
	const initialSecretHash = "bcrypt-hash-placeholder-initial"
	explicitClientID := "client-" + uuid.New().String()
	orgID := "org-" + uuid.New().String()
	sa := &schemas.Client{
		Name:                    "test_service_account_" + uuid.New().String(),
		Kind:                    "interactive",
		ClientID:                explicitClientID,
		ClientSecret:            initialSecretHash,
		AllowedScopes:           "read,write",
		RedirectURIs:            "https://app.example.com/callback,https://app.example.com/callback2",
		GrantTypes:              "authorization_code,refresh_token",
		TokenEndpointAuthMethod: "client_secret_basic",
		OrgID:                   &orgID,
		IsActive:                true,
	}

	created, err := provider.AddClient(ctx, sa)
	require.NoError(t, err)
	require.NotNil(t, created)

	clientID := created.AsAPIClient().ID
	require.NotEmpty(t, clientID)

	// GetClientByID MUST succeed when passed the id a real caller
	// actually has (the client_id from the create response / admin API),
	// not just the provider's raw internal representation.
	fetched, err := provider.GetClientByID(ctx, clientID)
	require.NoError(t, err, "GetClientByID must resolve the API-facing client_id")
	assert.Equal(t, sa.Name, fetched.Name)
	assert.Equal(t, "interactive", fetched.Kind, "Kind must round-trip through storage")
	assert.True(t, fetched.IsActive)
	assert.Equal(t, []string{"read", "write"}, fetched.ParsedAllowedScopes())

	// Interactive-kind registry columns must round-trip through storage.
	assert.Equal(t, explicitClientID, fetched.ClientID, "ClientID must round-trip through storage")
	assert.Equal(t, "https://app.example.com/callback,https://app.example.com/callback2", fetched.RedirectURIs, "RedirectURIs must round-trip")
	assert.Equal(t, "authorization_code,refresh_token", fetched.GrantTypes, "GrantTypes must round-trip")
	assert.Equal(t, "client_secret_basic", fetched.TokenEndpointAuthMethod, "TokenEndpointAuthMethod must round-trip")
	require.NotNil(t, fetched.OrgID, "OrgID must round-trip as a non-nil pointer")
	assert.Equal(t, orgID, *fetched.OrgID, "OrgID value must round-trip")

	// GetClientByClientID resolves by the public client_id (distinct from the
	// surrogate id) — the lookup the token endpoint and boot-time seed use.
	byClientID, err := provider.GetClientByClientID(ctx, explicitClientID)
	require.NoError(t, err, "GetClientByClientID must resolve the public client_id")
	assert.Equal(t, clientID, byClientID.AsAPIClient().ID, "GetClientByClientID must return the same client")
	assert.Equal(t, explicitClientID, byClientID.ClientID)

	// A second client with the same client_id must be rejected (unique client_id).
	_, dupErr := provider.AddClient(ctx, &schemas.Client{
		Name:         "dup_" + uuid.New().String(),
		Kind:         "interactive",
		ClientID:     explicitClientID,
		ClientSecret: initialSecretHash,
		IsActive:     true,
	})
	assert.Error(t, dupErr, "duplicate client_id must be rejected")
	// ClientSecret has json:"-" (kept out of API responses/logs) — a storage
	// provider that (de)serializes via encoding/json for persistence (e.g.
	// Couchbase) can silently drop it on write or read unless it routes
	// through a tag-aware helper. This must never regress: client_credentials
	// authentication depends on the stored hash actually being there.
	assert.Equal(t, initialSecretHash, fetched.ClientSecret, "ClientSecret must round-trip through storage")

	// Rotation (UpdateClient with a new hash) must persist too —
	// not just the initial Add.
	const rotatedSecretHash = "bcrypt-hash-placeholder-rotated"
	fetched.ClientSecret = rotatedSecretHash
	rotated, err := provider.UpdateClient(ctx, fetched)
	require.NoError(t, err)
	assert.Equal(t, rotatedSecretHash, rotated.ClientSecret)
	refetched, err := provider.GetClientByID(ctx, clientID)
	require.NoError(t, err)
	assert.Equal(t, rotatedSecretHash, refetched.ClientSecret, "rotated ClientSecret must persist, not silently no-op")
	fetched = refetched

	// UpdateClient: load-then-mutate, matching the service layer.
	fetched.IsActive = false
	updated, err := provider.UpdateClient(ctx, fetched)
	require.NoError(t, err)
	assert.False(t, updated.IsActive)

	// A TrustedIssuer bound to this service account (via the API-facing id,
	// exactly as the admin API stores it) must cascade-delete along with the
	// service account — not survive as an orphan.
	issuer, err := provider.AddTrustedIssuer(ctx, &schemas.TrustedIssuer{
		ClientID:      clientID,
		Name:          "test_issuer_" + uuid.New().String(),
		IssuerURL:     "https://issuer.example.com/" + uuid.New().String(),
		KeySourceType: "static_jwks_url",
		ExpectedAud:   "https://authorizer.example.com",
		SubjectClaim:  "sub",
		IssuerType:    "oidc",
		AuthMethod:    "jwt_assertion",
		IsActive:      true,
	})
	require.NoError(t, err)
	require.NotNil(t, issuer)

	// ListClients should include the created account, with its
	// (rotated) ClientSecret intact — this is the third read path that could
	// silently drop a json:"-" field.
	list, _, err := provider.ListClients(ctx, &model.Pagination{Limit: 50, Offset: 0})
	require.NoError(t, err)
	var foundInList *schemas.Client
	for _, s := range list {
		if s.AsAPIClient().ID == clientID {
			foundInList = s
			break
		}
	}
	require.NotNil(t, foundInList, "created service account should appear in ListClients")
	assert.Equal(t, rotatedSecretHash, foundInList.ClientSecret, "ListClients must not drop ClientSecret")

	// DeleteClient must cascade: the bound TrustedIssuer must be gone too.
	require.NoError(t, provider.DeleteClient(ctx, updated))

	_, err = provider.GetClientByID(ctx, clientID)
	assert.Error(t, err, "service account should be gone after delete")

	issuerAPIID := issuer.AsAPITrustedIssuer().ID
	_, err = provider.GetTrustedIssuerByID(ctx, issuerAPIID)
	assert.Error(t, err, "trusted issuer must be cascade-deleted with its parent service account, not orphaned")

	// Regression guard for a DynamoDB cascade bug that deleted the parent first
	// and swallowed the child-query error, orphaning trusted issuers that could
	// still authenticate client_assertion JWTs even after their SA was "deleted".
	remainingIssuers, _, err := provider.ListTrustedIssuers(ctx, clientID, &model.Pagination{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, len(remainingIssuers), "no orphaned trusted issuers should remain after cascade delete")
}

// testTrustedIssuerOperations exercises TrustedIssuer CRUD, again using the
// API-facing id for every lookup — see testClientOperations for why.
func testTrustedIssuerOperations(t *testing.T, ctx context.Context, provider Provider, dbType string) {
	sa, err := provider.AddClient(ctx, &schemas.Client{
		Name:          "test_ti_service_account_" + uuid.New().String(),
		AllowedScopes: "read",
		IsActive:      true,
	})
	require.NoError(t, err)
	saClientID := sa.AsAPIClient().ID

	issuerURL := "https://issuer.example.com/" + uuid.New().String()
	// A SPIFFE-typed row: IssuerType = spiffe_jwt plus the reused Phase-4
	// TokenReview fields must round-trip on every provider (no schema change was
	// introduced for SPIFFE/TokenReview — these are existing columns).
	issuer := &schemas.TrustedIssuer{
		ClientID:               saClientID,
		Name:                   "test_trusted_issuer_" + uuid.New().String(),
		IssuerURL:              issuerURL,
		KeySourceType:          "static_jwks_url",
		ExpectedAud:            "https://authorizer.example.com",
		SubjectClaim:           "sub",
		AllowedSubjects:        "system:serviceaccount:prod:payments,system:serviceaccount:prod:billing",
		IssuerType:             constants.IssuerTypeSPIFFEJWT,
		AuthMethod:             "jwt_assertion",
		EnableTokenReview:      true,
		KubernetesAPIServerURL: refs.NewStringRef("https://kube-apiserver.example.com:6443"),
		IsActive:               true,
	}

	created, err := provider.AddTrustedIssuer(ctx, issuer)
	require.NoError(t, err)
	require.NotNil(t, created)

	issuerID := created.AsAPITrustedIssuer().ID
	require.NotEmpty(t, issuerID)

	fetched, err := provider.GetTrustedIssuerByID(ctx, issuerID)
	require.NoError(t, err, "GetTrustedIssuerByID must resolve the API-facing id")
	assert.Equal(t, issuer.Name, fetched.Name)
	assert.Equal(t, issuerURL, fetched.IssuerURL)
	// AllowedSubjects (§5.2 C1 subject pin) must round-trip verbatim on every DB.
	assert.Equal(t, issuer.AllowedSubjects, fetched.AllowedSubjects)
	assert.Equal(t, []string{"system:serviceaccount:prod:payments", "system:serviceaccount:prod:billing"}, fetched.ParsedAllowedSubjects())
	// SPIFFE + TokenReview fields must round-trip on every provider.
	assert.Equal(t, constants.IssuerTypeSPIFFEJWT, fetched.IssuerType, "spiffe_jwt IssuerType must persist")
	assert.True(t, fetched.EnableTokenReview, "EnableTokenReview must persist")
	require.NotNil(t, fetched.KubernetesAPIServerURL)
	assert.Equal(t, "https://kube-apiserver.example.com:6443", *fetched.KubernetesAPIServerURL)

	fetchedByURL, err := provider.GetTrustedIssuerByIssuerURL(ctx, issuerURL)
	require.NoError(t, err)
	assert.Equal(t, fetched.Name, fetchedByURL.Name)
	assert.Equal(t, issuer.AllowedSubjects, fetchedByURL.AllowedSubjects)

	fetched.ExpectedAud = "https://updated-audience.example.com"
	fetched.AllowedSubjects = "system:serviceaccount:prod:payments"
	updated, err := provider.UpdateTrustedIssuer(ctx, fetched)
	require.NoError(t, err)
	assert.Equal(t, "https://updated-audience.example.com", updated.ExpectedAud)

	reFetched, err := provider.GetTrustedIssuerByID(ctx, issuerID)
	require.NoError(t, err)
	assert.Equal(t, "system:serviceaccount:prod:payments", reFetched.AllowedSubjects, "AllowedSubjects update must persist")

	// ListTrustedIssuers filtered by the API-facing service_account_id must
	// find this issuer.
	list, _, err := provider.ListTrustedIssuers(ctx, saClientID, &model.Pagination{Limit: 50, Offset: 0})
	require.NoError(t, err)
	foundInList := false
	for _, i := range list {
		if i.AsAPITrustedIssuer().ID == issuerID {
			foundInList = true
			break
		}
	}
	assert.True(t, foundInList, "created trusted issuer should appear in ListTrustedIssuers filtered by service_account_id")

	// IssuerURL must be unique per Authorizer instance: GetTrustedIssuerByIssuerURL
	// runs on every client_assertion validation and expects a single deterministic
	// match. A second issuer (even under a different service account) with the same
	// issuer_url must be rejected. Only the SQL providers (gorm uniqueIndex) and
	// Cassandra/ScyllaDB (check-then-insert guard) enforce this today; other NoSQL
	// providers have no equivalent guard yet, so scope the assertion to enforcing DBs.
	if dbType == constants.DbTypeSqlite || dbType == constants.DbTypePostgres || dbType == constants.DbTypeScyllaDB {
		sa2, err := provider.AddClient(ctx, &schemas.Client{
			Name:          "test_ti_service_account_dup_" + uuid.New().String(),
			AllowedScopes: "read",
			IsActive:      true,
		})
		require.NoError(t, err)
		_, err = provider.AddTrustedIssuer(ctx, &schemas.TrustedIssuer{
			ClientID:      sa2.AsAPIClient().ID,
			Name:          "test_trusted_issuer_dup_" + uuid.New().String(),
			IssuerURL:     issuerURL,
			KeySourceType: "static_jwks_url",
			ExpectedAud:   "https://authorizer.example.com",
			SubjectClaim:  "sub",
			IssuerType:    "oidc",
			AuthMethod:    "jwt_assertion",
			IsActive:      true,
		})
		assert.Error(t, err, "second trusted issuer with a duplicate issuer_url must be rejected")
		require.NoError(t, provider.DeleteClient(ctx, sa2))
	}

	require.NoError(t, provider.DeleteTrustedIssuer(ctx, updated))
	_, err = provider.GetTrustedIssuerByID(ctx, issuerID)
	assert.Error(t, err, "trusted issuer should be gone after delete")

	require.NoError(t, provider.DeleteClient(ctx, sa))
}

// testOrganizationOperations exercises Organization CRUD plus the cascade that
// removes an org's memberships on delete. Every lookup uses the API-facing id.
func testOrganizationOperations(t *testing.T, ctx context.Context, provider Provider) {
	displayName := "Acme Corporation"
	org := &schemas.Organization{
		Name:        "acme-" + uuid.New().String(),
		DisplayName: &displayName,
		Enabled:     true,
	}

	created, err := provider.AddOrganization(ctx, org)
	require.NoError(t, err)
	require.NotNil(t, created)

	orgID := created.AsAPIOrganization().ID
	require.NotEmpty(t, orgID)

	fetched, err := provider.GetOrganizationByID(ctx, orgID)
	require.NoError(t, err, "GetOrganizationByID must resolve the API-facing id")
	assert.Equal(t, org.Name, fetched.Name)
	require.NotNil(t, fetched.DisplayName)
	assert.Equal(t, displayName, *fetched.DisplayName)
	assert.True(t, fetched.Enabled)

	byName, err := provider.GetOrganizationByName(ctx, org.Name)
	require.NoError(t, err)
	assert.Equal(t, orgID, byName.AsAPIOrganization().ID, "GetOrganizationByName must resolve the same org")

	// Update: disable and rename display name — must persist (not silently no-op).
	newDisplay := "Acme Inc"
	fetched.Enabled = false
	fetched.DisplayName = &newDisplay
	updated, err := provider.UpdateOrganization(ctx, fetched)
	require.NoError(t, err)
	assert.False(t, updated.Enabled)
	refetched, err := provider.GetOrganizationByID(ctx, orgID)
	require.NoError(t, err)
	assert.False(t, refetched.Enabled, "disabled state must persist")
	require.NotNil(t, refetched.DisplayName)
	assert.Equal(t, newDisplay, *refetched.DisplayName)

	// A membership bound to this org (via the API-facing id) must cascade-delete
	// with the organization — not survive as an orphan.
	userID := uuid.New().String()
	membership, err := provider.AddOrgMembership(ctx, &schemas.OrgMembership{
		OrgID:  orgID,
		UserID: userID,
		Roles:  "admin",
	})
	require.NoError(t, err)
	require.NotNil(t, membership)

	// ListOrganizations should include the created org.
	list, _, err := provider.ListOrganizations(ctx, &model.Pagination{Limit: 50, Offset: 0})
	require.NoError(t, err)
	foundInList := false
	for _, o := range list {
		if o.AsAPIOrganization().ID == orgID {
			foundInList = true
			break
		}
	}
	assert.True(t, foundInList, "created organization should appear in ListOrganizations")

	// DeleteOrganization must cascade: the membership must be gone too.
	require.NoError(t, provider.DeleteOrganization(ctx, refetched))

	_, err = provider.GetOrganizationByID(ctx, orgID)
	assert.Error(t, err, "organization should be gone after delete")

	_, err = provider.GetOrgMembership(ctx, orgID, userID)
	assert.Error(t, err, "membership must be cascade-deleted with its organization, not orphaned")

	remaining, _, err := provider.ListOrgMembershipsByOrg(ctx, orgID, &model.Pagination{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 0, len(remaining), "no orphaned memberships should remain after cascade delete")
}

// testOrgMembershipOperations exercises membership uniqueness, cross-org role
// independence, and the by-user listing.
func testOrgMembershipOperations(t *testing.T, ctx context.Context, provider Provider) {
	orgA, err := provider.AddOrganization(ctx, &schemas.Organization{
		Name:    "org-a-" + uuid.New().String(),
		Enabled: true,
	})
	require.NoError(t, err)
	orgAID := orgA.AsAPIOrganization().ID

	orgB, err := provider.AddOrganization(ctx, &schemas.Organization{
		Name:    "org-b-" + uuid.New().String(),
		Enabled: true,
	})
	require.NoError(t, err)
	orgBID := orgB.AsAPIOrganization().ID

	userID := uuid.New().String()

	// Same user is admin in Org A and viewer in Org B — roles independent.
	memberA, err := provider.AddOrgMembership(ctx, &schemas.OrgMembership{
		OrgID:  orgAID,
		UserID: userID,
		Roles:  "admin",
	})
	require.NoError(t, err)
	_, err = provider.AddOrgMembership(ctx, &schemas.OrgMembership{
		OrgID:  orgBID,
		UserID: userID,
		Roles:  "viewer",
	})
	require.NoError(t, err)

	inA, err := provider.GetOrgMembership(ctx, orgAID, userID)
	require.NoError(t, err)
	assert.Equal(t, []string{"admin"}, inA.ParsedRoles())
	inB, err := provider.GetOrgMembership(ctx, orgBID, userID)
	require.NoError(t, err)
	assert.Equal(t, []string{"viewer"}, inB.ParsedRoles())

	// Duplicate membership: the same (org_id, user_id) must be rejected. Every
	// provider enforces this (unique index or check-then-insert guard).
	_, err = provider.AddOrgMembership(ctx, &schemas.OrgMembership{
		OrgID:  orgAID,
		UserID: userID,
		Roles:  "admin",
	})
	assert.Error(t, err, "a duplicate (org_id, user_id) membership must be rejected")

	// The user must be a member of exactly two organizations.
	byUser, _, err := provider.ListOrgMembershipsByUser(ctx, userID, &model.Pagination{Limit: 50, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 2, len(byUser), "user should have memberships in exactly two organizations")

	// Org A must list exactly this one member.
	byOrg, _, err := provider.ListOrgMembershipsByOrg(ctx, orgAID, &model.Pagination{Limit: 50, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 1, len(byOrg))
	assert.Equal(t, userID, byOrg[0].UserID)

	// UpdateOrgMembership: load-then-mutate the roles — must persist.
	inA.Roles = "admin,billing"
	_, err = provider.UpdateOrgMembership(ctx, inA)
	require.NoError(t, err)
	refetched, err := provider.GetOrgMembership(ctx, orgAID, userID)
	require.NoError(t, err)
	assert.Equal(t, []string{"admin", "billing"}, refetched.ParsedRoles(), "updated roles must persist")

	// Direct membership delete removes only the targeted membership.
	require.NoError(t, provider.DeleteOrgMembership(ctx, memberA))
	_, err = provider.GetOrgMembership(ctx, orgAID, userID)
	assert.Error(t, err, "membership should be gone after delete")
	_, err = provider.GetOrgMembership(ctx, orgBID, userID)
	assert.NoError(t, err, "deleting the Org A membership must not affect the Org B membership")

	// Multi-member org: two distinct users in ONE org. This exercises the path
	// where the DynamoDB GetOrgMembership filter+Limit bug (now fixed) returned a
	// false "not found" — Limit was applied before the user_id filter, so a lookup
	// for a member not scanned first missed it, silently allowing duplicate
	// memberships and breaking member removal. Single-member-per-org tests above
	// never hit this. Guards the regression under TEST_DBS=dynamodb.
	orgC, err := provider.AddOrganization(ctx, &schemas.Organization{
		Name:    "org-c-" + uuid.New().String(),
		Enabled: true,
	})
	require.NoError(t, err)
	orgCID := orgC.AsAPIOrganization().ID
	userX := uuid.New().String()
	userY := uuid.New().String()
	_, err = provider.AddOrgMembership(ctx, &schemas.OrgMembership{OrgID: orgCID, UserID: userX, Roles: "admin"})
	require.NoError(t, err)
	_, err = provider.AddOrgMembership(ctx, &schemas.OrgMembership{OrgID: orgCID, UserID: userY, Roles: "viewer"})
	require.NoError(t, err)
	gx, err := provider.GetOrgMembership(ctx, orgCID, userX)
	require.NoError(t, err, "first member must be found in a multi-member org")
	assert.Equal(t, userX, gx.UserID)
	gy, err := provider.GetOrgMembership(ctx, orgCID, userY)
	require.NoError(t, err, "second member must be found in a multi-member org (filter+Limit regression)")
	assert.Equal(t, userY, gy.UserID)
	_, err = provider.AddOrgMembership(ctx, &schemas.OrgMembership{OrgID: orgCID, UserID: userY, Roles: "viewer"})
	assert.Error(t, err, "duplicate membership in a multi-member org must be rejected")
	require.NoError(t, provider.DeleteOrganization(ctx, orgC))

	require.NoError(t, provider.DeleteOrganization(ctx, orgA))
	require.NoError(t, provider.DeleteOrganization(ctx, orgB))
}

// testOrgOIDCAndFederatedOperations exercises the sso_oidc TrustedIssuer fields
// (esp. the encrypted upstream secret round-trip + its json:"-" tag), the
// GetTrustedIssuerByOrgIDAndKind lookup, and FederatedIdentity Add/Get across
// every backend.
func testOrgOIDCAndFederatedOperations(t *testing.T, ctx context.Context, provider Provider, dbType string) {
	orgID := "org-" + uuid.New().String()
	issuerURL := "https://idp-" + uuid.New().String() + ".example.com"

	conn, err := provider.AddTrustedIssuer(ctx, &schemas.TrustedIssuer{
		Kind:               constants.TrustKindSSOOIDC,
		OrgID:              orgID,
		Name:               "conn_" + uuid.New().String(),
		IssuerURL:          issuerURL,
		KeySourceType:      "oidc_discovery",
		IssuerType:         "oidc",
		AuthMethod:         "jwt_assertion",
		SSOClientID:        "rp-client-id",
		SSOClientSecretEnc: "ENCRYPTED-SECRET-VALUE",
		SSOScopes:          "openid profile email",
		SSORedirectURI:     "https://authorizer.example.com/oauth/sso/x/callback",
		IsActive:           true,
	})
	require.NoError(t, err)
	connID := conn.AsAPITrustedIssuer().ID
	require.NotEmpty(t, connID)

	// Round-trip: every SSO field, especially the encrypted secret, must persist.
	fetched, err := provider.GetTrustedIssuerByID(ctx, connID)
	require.NoError(t, err)
	assert.Equal(t, constants.TrustKindSSOOIDC, fetched.Kind)
	assert.Equal(t, orgID, fetched.OrgID)
	assert.Equal(t, "rp-client-id", fetched.SSOClientID)
	assert.Equal(t, "ENCRYPTED-SECRET-VALUE", fetched.SSOClientSecretEnc, "upstream secret must persist across write->read on "+dbType)
	assert.Equal(t, "openid profile email", fetched.SSOScopes)
	assert.Equal(t, "https://authorizer.example.com/oauth/sso/x/callback", fetched.SSORedirectURI)

	// json:"-" — the secret must NEVER serialize into a JSON projection.
	raw, err := json.Marshal(fetched)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "ENCRYPTED-SECRET-VALUE", "sso_client_secret_enc must not serialize (json:\"-\")")
	assert.Contains(t, string(raw), issuerURL, "non-secret fields still serialize")

	// GetTrustedIssuerByOrgIDAndKind resolves the org's connection.
	byOrg, err := provider.GetTrustedIssuerByOrgIDAndKind(ctx, orgID, constants.TrustKindSSOOIDC)
	require.NoError(t, err)
	assert.Equal(t, connID, byOrg.AsAPITrustedIssuer().ID)

	// FederatedIdentity round-trip.
	sub := "sub-" + uuid.New().String()
	userID := "user-" + uuid.New().String()
	_, err = provider.AddFederatedIdentity(ctx, &schemas.FederatedIdentity{
		OrgID: orgID, Issuer: issuerURL, Subject: sub, UserID: userID,
	})
	require.NoError(t, err)

	fi, err := provider.GetFederatedIdentity(ctx, orgID, issuerURL, sub)
	require.NoError(t, err)
	assert.Equal(t, userID, fi.UserID)

	// A different subject at the same (org, issuer) is a distinct identity.
	_, err = provider.GetFederatedIdentity(ctx, orgID, issuerURL, "unknown-sub")
	assert.Error(t, err, "unknown federated identity must not resolve")
}

// testOrgSAMLOperations exercises the sso_saml TrustedIssuer fields (nullable
// SAML SP config) round-tripping across every backend, plus the
// GetTrustedIssuerByOrgIDAndKind lookup and an UpdateTrustedIssuer mutation.
func testOrgSAMLOperations(t *testing.T, ctx context.Context, provider Provider, dbType string) {
	orgID := "org-" + uuid.New().String()
	idpEntityID := "https://idp-" + uuid.New().String() + ".example.com/metadata"
	const certPEM = "-----BEGIN CERTIFICATE-----\nMIIBsomethingfakebutstable==\n-----END CERTIFICATE-----"
	const attrMap = `{"email":"mail","given_name":"gn"}`

	conn, err := provider.AddTrustedIssuer(ctx, &schemas.TrustedIssuer{
		Kind:                  constants.TrustKindSSOSAML,
		OrgID:                 orgID,
		Name:                  "saml_" + uuid.New().String(),
		IssuerURL:             idpEntityID,
		KeySourceType:         "saml_idp_certificate",
		IssuerType:            "saml",
		AuthMethod:            "saml_assertion",
		SAMLSSOURL:            refs.NewStringRef("https://idp.example.com/sso"),
		SAMLIDPCertPEM:        refs.NewStringRef(certPEM),
		SAMLSPEntityID:        refs.NewStringRef("https://authorizer.example.com/oauth/saml/x/metadata"),
		SAMLACSURL:            refs.NewStringRef("https://authorizer.example.com/oauth/saml/x/acs"),
		SAMLAttributeMapping:  refs.NewStringRef(attrMap),
		SAMLAllowIDPInitiated: true,
		IsActive:              true,
	})
	require.NoError(t, err)
	connID := conn.AsAPITrustedIssuer().ID
	require.NotEmpty(t, connID)

	// Round-trip: every SAML field must persist across write->read.
	fetched, err := provider.GetTrustedIssuerByID(ctx, connID)
	require.NoError(t, err)
	assert.Equal(t, constants.TrustKindSSOSAML, fetched.EffectiveKind())
	assert.Equal(t, orgID, fetched.OrgID)
	assert.Equal(t, idpEntityID, fetched.IssuerURL)
	require.NotNil(t, fetched.SAMLSSOURL)
	assert.Equal(t, "https://idp.example.com/sso", *fetched.SAMLSSOURL)
	require.NotNil(t, fetched.SAMLIDPCertPEM)
	assert.Equal(t, certPEM, *fetched.SAMLIDPCertPEM, "IdP certificate must persist on "+dbType)
	require.NotNil(t, fetched.SAMLSPEntityID)
	assert.Equal(t, "https://authorizer.example.com/oauth/saml/x/metadata", *fetched.SAMLSPEntityID)
	require.NotNil(t, fetched.SAMLACSURL)
	assert.Equal(t, "https://authorizer.example.com/oauth/saml/x/acs", *fetched.SAMLACSURL)
	require.NotNil(t, fetched.SAMLAttributeMapping)
	assert.Equal(t, attrMap, *fetched.SAMLAttributeMapping)
	assert.True(t, fetched.SAMLAllowIDPInitiated)

	// GetTrustedIssuerByOrgIDAndKind resolves the org's SAML connection.
	byOrg, err := provider.GetTrustedIssuerByOrgIDAndKind(ctx, orgID, constants.TrustKindSSOSAML)
	require.NoError(t, err)
	assert.Equal(t, connID, byOrg.AsAPITrustedIssuer().ID)

	// Update mutates SAML fields (load-then-mutate); the bool must flip to false.
	fetched.SAMLAllowIDPInitiated = false
	fetched.SAMLSSOURL = refs.NewStringRef("https://idp.example.com/sso2")
	_, err = provider.UpdateTrustedIssuer(ctx, fetched)
	require.NoError(t, err)
	reFetched, err := provider.GetTrustedIssuerByID(ctx, connID)
	require.NoError(t, err)
	assert.False(t, reFetched.SAMLAllowIDPInitiated, "SAMLAllowIDPInitiated must persist as false on "+dbType)
	require.NotNil(t, reFetched.SAMLSSOURL)
	assert.Equal(t, "https://idp.example.com/sso2", *reFetched.SAMLSSOURL)
}

// testScimEndpointOperations exercises ScimEndpoint CRUD including that the
// bcrypt token hash round-trips (a json:"-" secret must persist) and never
// serializes out, and that the org lookup resolves the same row.
func testScimEndpointOperations(t *testing.T, ctx context.Context, provider Provider) {
	org, err := provider.AddOrganization(ctx, &schemas.Organization{
		Name:    "scim-org-" + uuid.New().String(),
		Enabled: true,
	})
	require.NoError(t, err)
	orgID := org.AsAPIOrganization().ID

	const tokenHash = "$2a$12$abcdefghijklmnopqrstuvSCIMhashvaluethatmustroundtrip.xyz"
	created, err := provider.AddScimEndpoint(ctx, &schemas.ScimEndpoint{
		OrgID:     orgID,
		TokenHash: tokenHash,
		Enabled:   true,
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	// Use the API-facing id (bare key). ArangoDB returns created.ID as the full
	// "collection/key" handle; the runtime by-id lookup (GetScimEndpointByID) is
	// keyed on the bare id the token carries, exactly as AsAPIScimEndpoint exposes.
	endpointID := created.AsAPIScimEndpoint().ID
	require.NotEmpty(t, endpointID)

	// Lookup by id (the token-embedded key) must resolve the row and its hash.
	byID, err := provider.GetScimEndpointByID(ctx, endpointID)
	require.NoError(t, err)
	assert.Equal(t, orgID, byID.OrgID)
	assert.Equal(t, tokenHash, byID.TokenHash, "bcrypt token hash must round-trip")
	assert.True(t, byID.Enabled)

	// Lookup by org (admin surface) must resolve the same endpoint.
	byOrg, err := provider.GetScimEndpointByOrgID(ctx, orgID)
	require.NoError(t, err)
	assert.Equal(t, endpointID, byOrg.AsAPIScimEndpoint().ID)

	// The secret must never serialize out (json:"-").
	blob, err := json.Marshal(byID)
	require.NoError(t, err)
	assert.NotContains(t, string(blob), tokenHash, "token hash must not appear in JSON")

	// Rotate: update the hash (load-then-mutate) and confirm it persists.
	const rotated = "$2a$12$ROTATEDhashvaluethatmustalsoroundtrip.abcdefghijklmnop"
	byID.TokenHash = rotated
	_, err = provider.UpdateScimEndpoint(ctx, byID)
	require.NoError(t, err)
	afterRotate, err := provider.GetScimEndpointByID(ctx, endpointID)
	require.NoError(t, err)
	assert.Equal(t, rotated, afterRotate.TokenHash, "rotated hash must persist")

	// Delete.
	require.NoError(t, provider.DeleteScimEndpoint(ctx, afterRotate))
	_, err = provider.GetScimEndpointByID(ctx, endpointID)
	assert.Error(t, err, "endpoint should be gone after delete")

	require.NoError(t, provider.DeleteOrganization(ctx, org))
}

// testScimGroupOperations exercises ScimGroup CRUD: create, id + (org,displayName)
// lookups, displayName rename round-trip, external-id round-trip, and delete.
// Membership is NOT stored here (it lives in FGA), so this covers metadata only.
func testScimGroupOperations(t *testing.T, ctx context.Context, provider Provider) {
	org, err := provider.AddOrganization(ctx, &schemas.Organization{
		Name:    "scim-grp-org-" + uuid.New().String(),
		Enabled: true,
	})
	require.NoError(t, err)
	orgID := org.AsAPIOrganization().ID

	ext := orgID + ":ext-grp-1"
	created, err := provider.AddScimGroup(ctx, &schemas.ScimGroup{
		OrgID:       orgID,
		DisplayName: "Engineers",
		ExternalID:  &ext,
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	// Bare key is the universal by-id lookup key (arango returns ID as the full
	// "collection/key" handle; every other provider sets Key == ID == uuid).
	groupID := created.Key
	require.NotEmpty(t, groupID)

	byID, err := provider.GetScimGroupByID(ctx, groupID)
	require.NoError(t, err)
	assert.Equal(t, orgID, byID.OrgID)
	assert.Equal(t, "Engineers", byID.DisplayName)
	require.NotNil(t, byID.ExternalID)
	assert.Equal(t, ext, *byID.ExternalID)

	byName, err := provider.GetScimGroupByOrgAndDisplayName(ctx, orgID, "Engineers")
	require.NoError(t, err)
	assert.Equal(t, byID.OrgID, byName.OrgID)
	assert.Equal(t, "Engineers", byName.DisplayName)

	// Cross-org displayName lookup must NOT resolve this org's group.
	_, err = provider.GetScimGroupByOrgAndDisplayName(ctx, "another-org", "Engineers")
	assert.Error(t, err, "displayName lookup must be org-scoped")

	// Rename (load-then-mutate) round-trips.
	byID.DisplayName = "Platform"
	_, err = provider.UpdateScimGroup(ctx, byID)
	require.NoError(t, err)
	afterRename, err := provider.GetScimGroupByID(ctx, groupID)
	require.NoError(t, err)
	assert.Equal(t, "Platform", afterRename.DisplayName)

	require.NoError(t, provider.DeleteScimGroup(ctx, afterRename))
	_, err = provider.GetScimGroupByID(ctx, groupID)
	assert.Error(t, err, "group should be gone after delete")

	require.NoError(t, provider.DeleteOrganization(ctx, org))
}

// testOrgDomainOperations exercises the verified-domain table: atomic
// first-writer-wins on the domain primary key, same-org idempotency, listing,
// delete, and the org-delete cascade (which frees the domain for re-claim).
func testOrgDomainOperations(t *testing.T, ctx context.Context, provider Provider) {
	orgA, err := provider.AddOrganization(ctx, &schemas.Organization{
		Name:    "domain-org-a-" + uuid.New().String(),
		Enabled: true,
	})
	require.NoError(t, err)
	orgAID := orgA.AsAPIOrganization().ID
	orgB, err := provider.AddOrganization(ctx, &schemas.Organization{
		Name:    "domain-org-b-" + uuid.New().String(),
		Enabled: true,
	})
	require.NoError(t, err)
	orgBID := orgB.AsAPIOrganization().ID

	// unique per run so parallel DBs / reruns don't collide on the shared PK.
	domain := "acme-" + strings.ToLower(uuid.New().String()[:8]) + ".com"

	// Insert a verified domain for org A.
	created, err := provider.AddOrgDomain(ctx, &schemas.OrgDomain{
		ID:         domain,
		Domain:     domain,
		OrgID:      orgAID,
		VerifiedAt: time.Now().Unix(),
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, domain, created.Domain)

	// Reverse lookup (the HRD path) resolves the owning org.
	byDomain, err := provider.GetOrgDomainByDomain(ctx, domain)
	require.NoError(t, err)
	assert.Equal(t, orgAID, byDomain.OrgID)
	assert.Equal(t, domain, byDomain.Domain)
	assert.NotZero(t, byDomain.VerifiedAt)

	// Same-org re-add is idempotent: no error, no duplicate.
	again, err := provider.AddOrgDomain(ctx, &schemas.OrgDomain{
		ID:         domain,
		Domain:     domain,
		OrgID:      orgAID,
		VerifiedAt: time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.Equal(t, orgAID, again.OrgID)

	// First-writer-wins: org B claiming the SAME domain must be rejected
	// distinctly (this is the ATO invariant — one verified domain, one org).
	_, err = provider.AddOrgDomain(ctx, &schemas.OrgDomain{
		ID:         domain,
		Domain:     domain,
		OrgID:      orgBID,
		VerifiedAt: time.Now().Unix(),
	})
	require.ErrorIs(t, err, schemas.ErrOrgDomainConflict)

	// The row still points to org A after the losing write.
	stillA, err := provider.GetOrgDomainByDomain(ctx, domain)
	require.NoError(t, err)
	assert.Equal(t, orgAID, stillA.OrgID)

	// A second distinct domain for org A, to prove listing + cascade cover many.
	domain2 := "beta-" + strings.ToLower(uuid.New().String()[:8]) + ".com"
	_, err = provider.AddOrgDomain(ctx, &schemas.OrgDomain{
		ID:         domain2,
		Domain:     domain2,
		OrgID:      orgAID,
		VerifiedAt: time.Now().Unix(),
	})
	require.NoError(t, err)

	list, _, err := provider.ListOrgDomainsByOrg(ctx, orgAID, &model.Pagination{Limit: 50, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, list, 2)

	// Delete one domain; it disappears, the other remains.
	require.NoError(t, provider.DeleteOrgDomain(ctx, domain2))
	_, err = provider.GetOrgDomainByDomain(ctx, domain2)
	assert.Error(t, err)
	remaining, _, err := provider.ListOrgDomainsByOrg(ctx, orgAID, &model.Pagination{Limit: 50, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, remaining, 1)

	// Cascade: deleting org A must free its remaining verified domain (M1) —
	// otherwise the unique PK makes it permanently unclaimable.
	require.NoError(t, provider.DeleteOrganization(ctx, orgA))
	_, err = provider.GetOrgDomainByDomain(ctx, domain)
	assert.Error(t, err, "domain must be gone after its org is deleted")

	// The freed domain is now re-claimable by org B.
	reclaimed, err := provider.AddOrgDomain(ctx, &schemas.OrgDomain{
		ID:         domain,
		Domain:     domain,
		OrgID:      orgBID,
		VerifiedAt: time.Now().Unix(),
	})
	require.NoError(t, err)
	assert.Equal(t, orgBID, reclaimed.OrgID)

	require.NoError(t, provider.DeleteOrganization(ctx, orgB))
}

// testUserScimFields verifies the additive User fields (ExternalID, IsActive)
// persist and that GetUserByExternalID resolves by the org-namespaced key.
func testUserScimFields(t *testing.T, ctx context.Context, provider Provider) {
	orgID := uuid.New().String()
	email := "scim-user-" + uuid.New().String() + "@example.com"
	nsExt := orgID + ":okta-" + uuid.New().String()

	created, err := provider.AddUser(ctx, &schemas.User{
		ID:            uuid.New().String(),
		Email:         &email,
		SignupMethods: "scim",
		ExternalID:    &nsExt,
		IsActive:      true,
	})
	require.NoError(t, err)

	// Additive fields round-trip.
	fetched, err := provider.GetUserByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.ExternalID)
	assert.Equal(t, nsExt, *fetched.ExternalID)
	assert.True(t, fetched.IsActive, "IsActive must persist as true")

	// GetUserByExternalID resolves by the org-namespaced (orgID, rawExternalID).
	rawExt := nsExt[len(orgID)+1:]
	byExt, err := provider.GetUserByExternalID(ctx, orgID, rawExt)
	require.NoError(t, err)
	assert.Equal(t, created.ID, byExt.ID, "GetUserByExternalID must resolve the org-scoped user")

	// A different org must NOT resolve the same external id (H6 at the data layer).
	_, err = provider.GetUserByExternalID(ctx, "other-org", rawExt)
	assert.Error(t, err, "external id is namespaced per org — another org must not resolve it")

	// Deactivate: IsActive=false persists.
	fetched.IsActive = false
	_, err = provider.UpdateUser(ctx, fetched)
	require.NoError(t, err)
	afterDeactivate, err := provider.GetUserByID(ctx, created.ID)
	require.NoError(t, err)
	assert.False(t, afterDeactivate.IsActive, "IsActive=false must persist")
}

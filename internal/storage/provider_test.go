package storage

import (
	"context"
	"fmt"
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
		// Allow extra time for Couchbase container to become ready in tests
		cfg.CouchBaseWaitTimeout = 120
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
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
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
			provider, err := New(cfg, &Dependencies{
				Log: &logger,
			})
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
				testAuthenticatorOperations(t, ctx, provider)
			})

			t.Run("Email Template Operations", func(t *testing.T) {
				testEmailTemplateOperations(t, ctx, provider)
			})

			t.Run("OTP Operations", func(t *testing.T) {
				testOTPOperations(t, ctx, provider)
			})

			t.Run("Session Operations", func(t *testing.T) {
				testSessionOperations(t, ctx, provider)
			})

			t.Run("User Operations", func(t *testing.T) {
				testUserOperations(t, ctx, provider)
			})

			t.Run("Verification Request Operations", func(t *testing.T) {
				testVerificationRequestOperations(t, ctx, provider)
			})

			t.Run("Webhook Operations", func(t *testing.T) {
				testWebhookOperations(t, ctx, provider)
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

			t.Run("Resource Operations", func(t *testing.T) {
				testResourceOperations(t, ctx, provider)
			})

			t.Run("Scope Operations", func(t *testing.T) {
				testScopeOperations(t, ctx, provider)
			})

			t.Run("Policy Operations", func(t *testing.T) {
				testPolicyOperations(t, ctx, provider)
			})

			t.Run("Permission Operations", func(t *testing.T) {
				testPermissionOperations(t, ctx, provider)
			})

		})
	}
}

func testUserOperations(t *testing.T, ctx context.Context, provider Provider) {
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

	// Test UpdateUser
	fetchedUser.GivenName = refs.NewStringRef("Updated")
	updatedUser, err := provider.UpdateUser(ctx, fetchedUser)
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, "Updated", *updatedUser.GivenName)

	// Test ListUsers
	users, pagination, err := provider.ListUsers(ctx, &model.Pagination{
		Limit:  10,
		Offset: 0,
	})
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

	// Test UpdateUsers
	users, _, err = provider.ListUsers(ctx, &model.Pagination{
		Limit:  10,
		Offset: 0,
	})
	assert.NoError(t, err)
	assert.NotNil(t, users)
	assert.Greater(t, len(users), 0)
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

func testVerificationRequestOperations(t *testing.T, ctx context.Context, provider Provider) {
	vr := &schemas.VerificationRequest{
		Token:      uuid.New().String(),
		Email:      "test_" + uuid.New().String() + "@test.com",
		ExpiresAt:  time.Now().Add(24 * time.Hour).Unix(),
		Identifier: "email_verification",
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

func testWebhookOperations(t *testing.T, ctx context.Context, provider Provider) {
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

	// Test UpdateWebhook
	webhook.EndPoint = "https://test.com/webhook_updated"
	updated, err := provider.UpdateWebhook(ctx, webhook)
	assert.NoError(t, err)
	assert.Equal(t, webhook.EndPoint, updated.EndPoint)

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

func testEmailTemplateOperations(t *testing.T, ctx context.Context, provider Provider) {
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

	// Test UpdateEmailTemplate
	template.Template = "Updated template"
	updated, err := provider.UpdateEmailTemplate(ctx, template)
	assert.NoError(t, err)
	assert.Equal(t, template.Template, updated.Template)

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

func testAuthenticatorOperations(t *testing.T, ctx context.Context, provider Provider) {
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

	// Test UpdateAuthenticator
	auth.Secret = "updated_secret"
	updated, err := provider.UpdateAuthenticator(ctx, auth)
	assert.NoError(t, err)
	assert.Equal(t, "updated_secret", updated.Secret)
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

	// Extract just the user ID part for DeleteAllSessionTokensByUserID
	userIDPart := userId2[len("auth_provider:"):]
	err = provider.DeleteAllSessionTokensByUserID(ctx, userIDPart)
	assert.NoError(t, err)

	// Verify all sessions for user are deleted
	_, err = provider.GetSessionTokenByUserIDAndKey(ctx, userId2, "session_token_key")
	assert.Error(t, err)
	_, err = provider.GetSessionTokenByUserIDAndKey(ctx, userId2, "access_token_key")
	assert.Error(t, err)

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

// testResourceOperations covers all six Provider methods for authorization resources:
// AddResource, GetResourceByID, GetResourceByName, UpdateResource, ListResources, DeleteResource.
func testResourceOperations(t *testing.T, ctx context.Context, provider Provider) {
	t.Helper()

	t.Run("add and get by id", func(t *testing.T) {
		id := uuid.New().String()
		r := &schemas.Resource{
			ID:          id,
			Key:         id,
			Name:        "res-add-get-" + id[:8],
			Description: "test resource",
		}
		created, err := provider.AddResource(ctx, r)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, r.Name, created.Name)

		fetched, err := provider.GetResourceByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, r.Name, fetched.Name)

		// cleanup
		require.NoError(t, provider.DeleteResource(ctx, id))
	})

	t.Run("add and get by name", func(t *testing.T) {
		id := uuid.New().String()
		name := "res-byname-" + id[:8]
		r := &schemas.Resource{ID: id, Key: id, Name: name, Description: "by-name test"}
		_, err := provider.AddResource(ctx, r)
		require.NoError(t, err)

		fetched, err := provider.GetResourceByName(ctx, name)
		require.NoError(t, err)
		assert.Equal(t, name, fetched.Name)

		require.NoError(t, provider.DeleteResource(ctx, id))
	})

	t.Run("update mutates persisted fields", func(t *testing.T) {
		id := uuid.New().String()
		r := &schemas.Resource{ID: id, Key: id, Name: "res-update-" + id[:8], Description: "original"}
		created, err := provider.AddResource(ctx, r)
		require.NoError(t, err)

		created.Description = "updated description"
		updated, err := provider.UpdateResource(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, "updated description", updated.Description)

		refetched, err := provider.GetResourceByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "updated description", refetched.Description)

		require.NoError(t, provider.DeleteResource(ctx, id))
	})

	t.Run("delete removes the row", func(t *testing.T) {
		id := uuid.New().String()
		r := &schemas.Resource{ID: id, Key: id, Name: "res-delete-" + id[:8]}
		_, err := provider.AddResource(ctx, r)
		require.NoError(t, err)

		require.NoError(t, provider.DeleteResource(ctx, id))

		_, err = provider.GetResourceByID(ctx, id)
		assert.Error(t, err, "GetResourceByID should return error after deletion")
	})

	t.Run("list returns inserted rows with correct pagination", func(t *testing.T) {
		// Insert 3 uniquely-named resources.
		suffix := uuid.New().String()[:8]
		ids := make([]string, 3)
		for i := range ids {
			id := uuid.New().String()
			ids[i] = id
			name := fmt.Sprintf("res-list-%s-%d", suffix, i)
			r := &schemas.Resource{ID: id, Key: id, Name: name}
			_, err := provider.AddResource(ctx, r)
			require.NoError(t, err)
		}

		// First page: limit 2, offset 0.
		pag1 := &model.Pagination{Limit: 2, Offset: 0}
		items1, retPag1, err := provider.ListResources(ctx, pag1)
		require.NoError(t, err)
		require.NotNil(t, retPag1)
		assert.GreaterOrEqual(t, retPag1.Total, int64(3))
		assert.LessOrEqual(t, len(items1), 2)

		// Second page: limit 2, offset 2.
		pag2 := &model.Pagination{Limit: 2, Offset: 2}
		items2, retPag2, err := provider.ListResources(ctx, pag2)
		require.NoError(t, err)
		require.NotNil(t, retPag2)
		assert.GreaterOrEqual(t, len(items2), 0)

		// cleanup
		for _, id := range ids {
			_ = provider.DeleteResource(ctx, id)
		}
	})

	t.Run("list does not mutate caller pagination pointer", func(t *testing.T) {
		pag := &model.Pagination{Limit: 10, Offset: 0}
		_, retPag, err := provider.ListResources(ctx, pag)
		require.NoError(t, err)
		assert.NotSame(t, pag, retPag, "ListResources should return a new pagination object")
	})

	t.Run("add duplicate name returns error", func(t *testing.T) {
		if strings.Contains(t.Name(), constants.DbTypeScyllaDB) {
			t.Skip("Cassandra/ScyllaDB does not enforce uniqueness constraints on non-partition-key columns")
		}
		id1 := uuid.New().String()
		name := "res-dup-" + id1[:8]
		r1 := &schemas.Resource{ID: id1, Key: id1, Name: name}
		_, err := provider.AddResource(ctx, r1)
		require.NoError(t, err)

		id2 := uuid.New().String()
		r2 := &schemas.Resource{ID: id2, Key: id2, Name: name}
		_, err = provider.AddResource(ctx, r2)
		assert.Error(t, err, "adding a resource with a duplicate name should fail")

		require.NoError(t, provider.DeleteResource(ctx, id1))
	})
}

// testScopeOperations covers all six Provider methods for authorization scopes:
// AddScope, GetScopeByID, GetScopeByName, UpdateScope, ListScopes, DeleteScope.
func testScopeOperations(t *testing.T, ctx context.Context, provider Provider) {
	t.Helper()

	t.Run("add and get by id", func(t *testing.T) {
		id := uuid.New().String()
		s := &schemas.Scope{
			ID:          id,
			Key:         id,
			Name:        "scope-add-" + id[:8],
			Description: "test scope",
		}
		created, err := provider.AddScope(ctx, s)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, s.Name, created.Name)

		fetched, err := provider.GetScopeByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, s.Name, fetched.Name)

		require.NoError(t, provider.DeleteScope(ctx, id))
	})

	t.Run("add and get by name", func(t *testing.T) {
		id := uuid.New().String()
		name := "scope-byname-" + id[:8]
		s := &schemas.Scope{ID: id, Key: id, Name: name, Description: "by-name test"}
		_, err := provider.AddScope(ctx, s)
		require.NoError(t, err)

		fetched, err := provider.GetScopeByName(ctx, name)
		require.NoError(t, err)
		assert.Equal(t, name, fetched.Name)

		require.NoError(t, provider.DeleteScope(ctx, id))
	})

	t.Run("update mutates persisted fields", func(t *testing.T) {
		id := uuid.New().String()
		s := &schemas.Scope{ID: id, Key: id, Name: "scope-update-" + id[:8], Description: "original"}
		created, err := provider.AddScope(ctx, s)
		require.NoError(t, err)

		created.Description = "updated scope description"
		updated, err := provider.UpdateScope(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, "updated scope description", updated.Description)

		refetched, err := provider.GetScopeByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "updated scope description", refetched.Description)

		require.NoError(t, provider.DeleteScope(ctx, id))
	})

	t.Run("delete removes the row", func(t *testing.T) {
		id := uuid.New().String()
		s := &schemas.Scope{ID: id, Key: id, Name: "scope-delete-" + id[:8]}
		_, err := provider.AddScope(ctx, s)
		require.NoError(t, err)

		require.NoError(t, provider.DeleteScope(ctx, id))

		_, err = provider.GetScopeByID(ctx, id)
		assert.Error(t, err, "GetScopeByID should return error after deletion")
	})

	t.Run("list returns inserted rows with correct pagination", func(t *testing.T) {
		suffix := uuid.New().String()[:8]
		ids := make([]string, 3)
		for i := range ids {
			id := uuid.New().String()
			ids[i] = id
			name := fmt.Sprintf("scope-list-%s-%d", suffix, i)
			s := &schemas.Scope{ID: id, Key: id, Name: name}
			_, err := provider.AddScope(ctx, s)
			require.NoError(t, err)
		}

		pag1 := &model.Pagination{Limit: 2, Offset: 0}
		items1, retPag1, err := provider.ListScopes(ctx, pag1)
		require.NoError(t, err)
		require.NotNil(t, retPag1)
		assert.GreaterOrEqual(t, retPag1.Total, int64(3))
		assert.LessOrEqual(t, len(items1), 2)

		pag2 := &model.Pagination{Limit: 2, Offset: 2}
		items2, retPag2, err := provider.ListScopes(ctx, pag2)
		require.NoError(t, err)
		require.NotNil(t, retPag2)
		assert.GreaterOrEqual(t, len(items2), 0)

		for _, id := range ids {
			_ = provider.DeleteScope(ctx, id)
		}
	})

	t.Run("list does not mutate caller pagination pointer", func(t *testing.T) {
		pag := &model.Pagination{Limit: 10, Offset: 0}
		_, retPag, err := provider.ListScopes(ctx, pag)
		require.NoError(t, err)
		assert.NotSame(t, pag, retPag, "ListScopes should return a new pagination object")
	})

	t.Run("add duplicate name returns error", func(t *testing.T) {
		if strings.Contains(t.Name(), constants.DbTypeScyllaDB) {
			t.Skip("Cassandra/ScyllaDB does not enforce uniqueness constraints on non-partition-key columns")
		}
		id1 := uuid.New().String()
		name := "scope-dup-" + id1[:8]
		s1 := &schemas.Scope{ID: id1, Key: id1, Name: name}
		_, err := provider.AddScope(ctx, s1)
		require.NoError(t, err)

		id2 := uuid.New().String()
		s2 := &schemas.Scope{ID: id2, Key: id2, Name: name}
		_, err = provider.AddScope(ctx, s2)
		assert.Error(t, err, "adding a scope with a duplicate name should fail")

		require.NoError(t, provider.DeleteScope(ctx, id1))
	})
}

// testPolicyOperations covers all eight Provider methods for authorization policies:
// AddPolicy, GetPolicyByID, UpdatePolicy, ListPolicies, DeletePolicy,
// AddPolicyTarget, GetPolicyTargets, DeletePolicyTargetsByPolicyID.
func testPolicyOperations(t *testing.T, ctx context.Context, provider Provider) {
	t.Helper()

	t.Run("add and get by id", func(t *testing.T) {
		id := uuid.New().String()
		p := &schemas.Policy{
			ID:               id,
			Key:              id,
			Name:             "pol-add-" + id[:8],
			Description:      "test policy",
			Type:             constants.PolicyTypeRole,
			Logic:            constants.PolicyLogicPositive,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		created, err := provider.AddPolicy(ctx, p)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, p.Name, created.Name)
		assert.Equal(t, constants.PolicyTypeRole, created.Type)

		fetched, err := provider.GetPolicyByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, p.Name, fetched.Name)
		assert.Equal(t, constants.PolicyLogicPositive, fetched.Logic)

		require.NoError(t, provider.DeletePolicy(ctx, id))
	})

	t.Run("update mutates persisted fields", func(t *testing.T) {
		id := uuid.New().String()
		p := &schemas.Policy{
			ID:               id,
			Key:              id,
			Name:             "pol-update-" + id[:8],
			Type:             constants.PolicyTypeRole,
			Logic:            constants.PolicyLogicPositive,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		created, err := provider.AddPolicy(ctx, p)
		require.NoError(t, err)

		created.Description = "updated policy description"
		created.Logic = constants.PolicyLogicNegative
		updated, err := provider.UpdatePolicy(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, "updated policy description", updated.Description)
		assert.Equal(t, constants.PolicyLogicNegative, updated.Logic)

		refetched, err := provider.GetPolicyByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, constants.PolicyLogicNegative, refetched.Logic)

		require.NoError(t, provider.DeletePolicy(ctx, id))
	})

	t.Run("delete removes the row", func(t *testing.T) {
		id := uuid.New().String()
		p := &schemas.Policy{
			ID:               id,
			Key:              id,
			Name:             "pol-delete-" + id[:8],
			Type:             constants.PolicyTypeUser,
			Logic:            constants.PolicyLogicPositive,
			DecisionStrategy: constants.DecisionStrategyUnanimous,
		}
		_, err := provider.AddPolicy(ctx, p)
		require.NoError(t, err)

		require.NoError(t, provider.DeletePolicy(ctx, id))

		_, err = provider.GetPolicyByID(ctx, id)
		assert.Error(t, err, "GetPolicyByID should return error after deletion")
	})

	t.Run("list returns inserted rows with correct pagination", func(t *testing.T) {
		suffix := uuid.New().String()[:8]
		ids := make([]string, 3)
		for i := range ids {
			id := uuid.New().String()
			ids[i] = id
			name := fmt.Sprintf("pol-list-%s-%d", suffix, i)
			p := &schemas.Policy{
				ID:               id,
				Key:              id,
				Name:             name,
				Type:             constants.PolicyTypeRole,
				Logic:            constants.PolicyLogicPositive,
				DecisionStrategy: constants.DecisionStrategyAffirmative,
			}
			_, err := provider.AddPolicy(ctx, p)
			require.NoError(t, err)
		}

		pag1 := &model.Pagination{Limit: 2, Offset: 0}
		items1, retPag1, err := provider.ListPolicies(ctx, pag1)
		require.NoError(t, err)
		require.NotNil(t, retPag1)
		assert.GreaterOrEqual(t, retPag1.Total, int64(3))
		assert.LessOrEqual(t, len(items1), 2)

		pag2 := &model.Pagination{Limit: 2, Offset: 2}
		items2, retPag2, err := provider.ListPolicies(ctx, pag2)
		require.NoError(t, err)
		require.NotNil(t, retPag2)
		assert.GreaterOrEqual(t, len(items2), 0)

		for _, id := range ids {
			_ = provider.DeletePolicy(ctx, id)
		}
	})

	t.Run("list does not mutate caller pagination pointer", func(t *testing.T) {
		pag := &model.Pagination{Limit: 10, Offset: 0}
		_, retPag, err := provider.ListPolicies(ctx, pag)
		require.NoError(t, err)
		assert.NotSame(t, pag, retPag, "ListPolicies should return a new pagination object")
	})

	t.Run("policy targets add get and delete", func(t *testing.T) {
		// Create parent policy.
		polID := uuid.New().String()
		pol := &schemas.Policy{
			ID:               polID,
			Key:              polID,
			Name:             "pol-targets-" + polID[:8],
			Type:             constants.PolicyTypeRole,
			Logic:            constants.PolicyLogicPositive,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		_, err := provider.AddPolicy(ctx, pol)
		require.NoError(t, err)

		// Add two targets.
		t1ID := uuid.New().String()
		t2ID := uuid.New().String()
		tgt1 := &schemas.PolicyTarget{
			ID:          t1ID,
			Key:         t1ID,
			PolicyID:    polID,
			TargetType:  "role",
			TargetValue: "editor",
			CreatedAt:   time.Now().Unix(),
		}
		tgt2 := &schemas.PolicyTarget{
			ID:          t2ID,
			Key:         t2ID,
			PolicyID:    polID,
			TargetType:  "role",
			TargetValue: "admin",
			CreatedAt:   time.Now().Unix(),
		}
		_, err = provider.AddPolicyTarget(ctx, tgt1)
		require.NoError(t, err)
		_, err = provider.AddPolicyTarget(ctx, tgt2)
		require.NoError(t, err)

		// GetPolicyTargets should return both.
		targets, err := provider.GetPolicyTargets(ctx, polID)
		require.NoError(t, err)
		assert.Len(t, targets, 2, "expected 2 policy targets")

		// DeletePolicyTargetsByPolicyID removes all targets.
		require.NoError(t, provider.DeletePolicyTargetsByPolicyID(ctx, polID))

		targets, err = provider.GetPolicyTargets(ctx, polID)
		require.NoError(t, err)
		assert.Empty(t, targets, "targets should be empty after DeletePolicyTargetsByPolicyID")

		// cleanup policy
		require.NoError(t, provider.DeletePolicy(ctx, polID))
	})
}

// testPermissionOperations covers all Provider methods for authorization permissions
// and their join tables:
// AddPermission, GetPermissionByID, UpdatePermission, ListPermissions, DeletePermission,
// AddPermissionScope, GetPermissionScopes, DeletePermissionScopesByPermissionID,
// AddPermissionPolicy, GetPermissionPolicies, DeletePermissionPoliciesByPermissionID,
// GetPermissionsForResourceScope.
func testPermissionOperations(t *testing.T, ctx context.Context, provider Provider) {
	t.Helper()

	// Helper to create a throwaway resource for permission tests.
	newResource := func(t *testing.T, nameSuffix string) *schemas.Resource {
		t.Helper()
		id := uuid.New().String()
		r := &schemas.Resource{ID: id, Key: id, Name: "perm-res-" + nameSuffix + "-" + id[:8]}
		created, err := provider.AddResource(ctx, r)
		require.NoError(t, err)
		return created
	}

	t.Run("add and get by id", func(t *testing.T) {
		res := newResource(t, "add")
		id := uuid.New().String()
		perm := &schemas.Permission{
			ID:               id,
			Key:              id,
			Name:             "perm-add-" + id[:8],
			Description:      "test permission",
			ResourceID:       res.ID,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		created, err := provider.AddPermission(ctx, perm)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, perm.Name, created.Name)
		assert.Equal(t, res.ID, created.ResourceID)

		fetched, err := provider.GetPermissionByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, perm.Name, fetched.Name)

		// cleanup: permission first (no join rows), then resource
		require.NoError(t, provider.DeletePermission(ctx, id))
		require.NoError(t, provider.DeleteResource(ctx, res.ID))
	})

	t.Run("update mutates persisted fields", func(t *testing.T) {
		res := newResource(t, "upd")
		id := uuid.New().String()
		perm := &schemas.Permission{
			ID:               id,
			Key:              id,
			Name:             "perm-upd-" + id[:8],
			ResourceID:       res.ID,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		created, err := provider.AddPermission(ctx, perm)
		require.NoError(t, err)

		created.Description = "updated permission description"
		created.DecisionStrategy = constants.DecisionStrategyUnanimous
		updated, err := provider.UpdatePermission(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, "updated permission description", updated.Description)
		assert.Equal(t, constants.DecisionStrategyUnanimous, updated.DecisionStrategy)

		refetched, err := provider.GetPermissionByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, constants.DecisionStrategyUnanimous, refetched.DecisionStrategy)

		require.NoError(t, provider.DeletePermission(ctx, id))
		require.NoError(t, provider.DeleteResource(ctx, res.ID))
	})

	t.Run("delete removes the row", func(t *testing.T) {
		res := newResource(t, "del")
		id := uuid.New().String()
		perm := &schemas.Permission{
			ID:               id,
			Key:              id,
			Name:             "perm-del-" + id[:8],
			ResourceID:       res.ID,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		_, err := provider.AddPermission(ctx, perm)
		require.NoError(t, err)

		require.NoError(t, provider.DeletePermission(ctx, id))

		_, err = provider.GetPermissionByID(ctx, id)
		assert.Error(t, err, "GetPermissionByID should return error after deletion")

		require.NoError(t, provider.DeleteResource(ctx, res.ID))
	})

	t.Run("list returns inserted rows with correct pagination", func(t *testing.T) {
		res := newResource(t, "list")
		suffix := uuid.New().String()[:8]
		ids := make([]string, 3)
		for i := range ids {
			id := uuid.New().String()
			ids[i] = id
			name := fmt.Sprintf("perm-list-%s-%d", suffix, i)
			perm := &schemas.Permission{
				ID:               id,
				Key:              id,
				Name:             name,
				ResourceID:       res.ID,
				DecisionStrategy: constants.DecisionStrategyAffirmative,
			}
			_, err := provider.AddPermission(ctx, perm)
			require.NoError(t, err)
		}

		pag1 := &model.Pagination{Limit: 2, Offset: 0}
		items1, retPag1, err := provider.ListPermissions(ctx, pag1)
		require.NoError(t, err)
		require.NotNil(t, retPag1)
		assert.GreaterOrEqual(t, retPag1.Total, int64(3))
		assert.LessOrEqual(t, len(items1), 2)

		pag2 := &model.Pagination{Limit: 2, Offset: 2}
		items2, retPag2, err := provider.ListPermissions(ctx, pag2)
		require.NoError(t, err)
		require.NotNil(t, retPag2)
		assert.GreaterOrEqual(t, len(items2), 0)

		// cleanup: permissions first, then resource
		for _, id := range ids {
			_ = provider.DeletePermission(ctx, id)
		}
		require.NoError(t, provider.DeleteResource(ctx, res.ID))
	})

	t.Run("list does not mutate caller pagination pointer", func(t *testing.T) {
		pag := &model.Pagination{Limit: 10, Offset: 0}
		_, retPag, err := provider.ListPermissions(ctx, pag)
		require.NoError(t, err)
		assert.NotSame(t, pag, retPag, "ListPermissions should return a new pagination object")
	})

	t.Run("permission scopes add get and delete", func(t *testing.T) {
		res := newResource(t, "ps")
		permID := uuid.New().String()
		perm := &schemas.Permission{
			ID:               permID,
			Key:              permID,
			Name:             "perm-ps-" + permID[:8],
			ResourceID:       res.ID,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		_, err := provider.AddPermission(ctx, perm)
		require.NoError(t, err)

		// Create two scopes and link them.
		scopeID1 := uuid.New().String()
		scope1 := &schemas.Scope{ID: scopeID1, Key: scopeID1, Name: "scope-ps1-" + scopeID1[:8]}
		_, err = provider.AddScope(ctx, scope1)
		require.NoError(t, err)

		scopeID2 := uuid.New().String()
		scope2 := &schemas.Scope{ID: scopeID2, Key: scopeID2, Name: "scope-ps2-" + scopeID2[:8]}
		_, err = provider.AddScope(ctx, scope2)
		require.NoError(t, err)

		ps1ID := uuid.New().String()
		ps1 := &schemas.PermissionScope{
			ID:           ps1ID,
			Key:          ps1ID,
			PermissionID: permID,
			ScopeID:      scopeID1,
			CreatedAt:    time.Now().Unix(),
		}
		ps2ID := uuid.New().String()
		ps2 := &schemas.PermissionScope{
			ID:           ps2ID,
			Key:          ps2ID,
			PermissionID: permID,
			ScopeID:      scopeID2,
			CreatedAt:    time.Now().Unix(),
		}
		_, err = provider.AddPermissionScope(ctx, ps1)
		require.NoError(t, err)
		_, err = provider.AddPermissionScope(ctx, ps2)
		require.NoError(t, err)

		psLinks, err := provider.GetPermissionScopes(ctx, permID)
		require.NoError(t, err)
		assert.Len(t, psLinks, 2, "expected 2 permission-scope links")

		require.NoError(t, provider.DeletePermissionScopesByPermissionID(ctx, permID))

		psLinks, err = provider.GetPermissionScopes(ctx, permID)
		require.NoError(t, err)
		assert.Empty(t, psLinks, "permission scopes should be empty after delete")

		// cleanup
		require.NoError(t, provider.DeletePermission(ctx, permID))
		require.NoError(t, provider.DeleteScope(ctx, scopeID1))
		require.NoError(t, provider.DeleteScope(ctx, scopeID2))
		require.NoError(t, provider.DeleteResource(ctx, res.ID))
	})

	t.Run("permission policies add get and delete", func(t *testing.T) {
		res := newResource(t, "pp")
		permID := uuid.New().String()
		perm := &schemas.Permission{
			ID:               permID,
			Key:              permID,
			Name:             "perm-pp-" + permID[:8],
			ResourceID:       res.ID,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		_, err := provider.AddPermission(ctx, perm)
		require.NoError(t, err)

		// Create two policies and link them.
		polID1 := uuid.New().String()
		pol1 := &schemas.Policy{
			ID:               polID1,
			Key:              polID1,
			Name:             "pol-pp1-" + polID1[:8],
			Type:             constants.PolicyTypeRole,
			Logic:            constants.PolicyLogicPositive,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		_, err = provider.AddPolicy(ctx, pol1)
		require.NoError(t, err)

		polID2 := uuid.New().String()
		pol2 := &schemas.Policy{
			ID:               polID2,
			Key:              polID2,
			Name:             "pol-pp2-" + polID2[:8],
			Type:             constants.PolicyTypeUser,
			Logic:            constants.PolicyLogicPositive,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		_, err = provider.AddPolicy(ctx, pol2)
		require.NoError(t, err)

		pp1ID := uuid.New().String()
		pp1 := &schemas.PermissionPolicy{
			ID:           pp1ID,
			Key:          pp1ID,
			PermissionID: permID,
			PolicyID:     polID1,
			CreatedAt:    time.Now().Unix(),
		}
		pp2ID := uuid.New().String()
		pp2 := &schemas.PermissionPolicy{
			ID:           pp2ID,
			Key:          pp2ID,
			PermissionID: permID,
			PolicyID:     polID2,
			CreatedAt:    time.Now().Unix(),
		}
		_, err = provider.AddPermissionPolicy(ctx, pp1)
		require.NoError(t, err)
		_, err = provider.AddPermissionPolicy(ctx, pp2)
		require.NoError(t, err)

		ppLinks, err := provider.GetPermissionPolicies(ctx, permID)
		require.NoError(t, err)
		assert.Len(t, ppLinks, 2, "expected 2 permission-policy links")

		require.NoError(t, provider.DeletePermissionPoliciesByPermissionID(ctx, permID))

		ppLinks, err = provider.GetPermissionPolicies(ctx, permID)
		require.NoError(t, err)
		assert.Empty(t, ppLinks, "permission policies should be empty after delete")

		// cleanup
		require.NoError(t, provider.DeletePermission(ctx, permID))
		require.NoError(t, provider.DeletePolicy(ctx, polID1))
		require.NoError(t, provider.DeletePolicy(ctx, polID2))
		require.NoError(t, provider.DeleteResource(ctx, res.ID))
	})

	t.Run("GetPermissionsForResourceScope evaluator hot-path", func(t *testing.T) {
		// Seed: resource + scope + policy (with one target) + permission linking all three.
		resID := uuid.New().String()
		suffix := resID[:8]
		resource := &schemas.Resource{ID: resID, Key: resID, Name: "evalres-" + suffix}
		_, err := provider.AddResource(ctx, resource)
		require.NoError(t, err)

		scopeID := uuid.New().String()
		scope := &schemas.Scope{ID: scopeID, Key: scopeID, Name: "evalscope-" + suffix}
		_, err = provider.AddScope(ctx, scope)
		require.NoError(t, err)

		polID := uuid.New().String()
		policy := &schemas.Policy{
			ID:               polID,
			Key:              polID,
			Name:             "evalpol-" + suffix,
			Type:             constants.PolicyTypeRole,
			Logic:            constants.PolicyLogicPositive,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		_, err = provider.AddPolicy(ctx, policy)
		require.NoError(t, err)

		tgtID := uuid.New().String()
		tgt := &schemas.PolicyTarget{
			ID:          tgtID,
			Key:         tgtID,
			PolicyID:    polID,
			TargetType:  "role",
			TargetValue: "editor",
			CreatedAt:   time.Now().Unix(),
		}
		_, err = provider.AddPolicyTarget(ctx, tgt)
		require.NoError(t, err)

		permID := uuid.New().String()
		perm := &schemas.Permission{
			ID:               permID,
			Key:              permID,
			Name:             "evalperm-" + suffix,
			ResourceID:       resID,
			DecisionStrategy: constants.DecisionStrategyAffirmative,
		}
		_, err = provider.AddPermission(ctx, perm)
		require.NoError(t, err)

		// Link scope to permission.
		psID := uuid.New().String()
		ps := &schemas.PermissionScope{
			ID:           psID,
			Key:          psID,
			PermissionID: permID,
			ScopeID:      scopeID,
			CreatedAt:    time.Now().Unix(),
		}
		_, err = provider.AddPermissionScope(ctx, ps)
		require.NoError(t, err)

		// Link policy to permission.
		ppID := uuid.New().String()
		pp := &schemas.PermissionPolicy{
			ID:           ppID,
			Key:          ppID,
			PermissionID: permID,
			PolicyID:     polID,
			CreatedAt:    time.Now().Unix(),
		}
		_, err = provider.AddPermissionPolicy(ctx, pp)
		require.NoError(t, err)

		// Query the evaluator hot-path by resource name and scope name.
		results, err := provider.GetPermissionsForResourceScope(ctx, resource.Name, scope.Name)
		require.NoError(t, err)
		require.NotEmpty(t, results, "expected at least one PermissionWithPolicies")

		// Find our seeded permission in the results (other tests may have left rows).
		var found *schemas.PermissionWithPolicies
		for _, r := range results {
			if r.PermissionID == permID {
				found = r
				break
			}
		}
		require.NotNil(t, found, "seeded permission not found in GetPermissionsForResourceScope result")
		assert.Equal(t, perm.Name, found.PermissionName)
		assert.Equal(t, constants.DecisionStrategyAffirmative, found.DecisionStrategy)
		require.NotEmpty(t, found.Policies, "expected at least one policy in the result")

		var foundPol *schemas.PolicyWithTargets
		for i := range found.Policies {
			if found.Policies[i].PolicyID == polID {
				foundPol = &found.Policies[i]
				break
			}
		}
		require.NotNil(t, foundPol, "seeded policy not found in PermissionWithPolicies.Policies")
		assert.Equal(t, policy.Name, foundPol.PolicyName)
		assert.Equal(t, constants.PolicyTypeRole, foundPol.Type)
		require.NotEmpty(t, foundPol.Targets, "expected policy target in result")
		assert.Equal(t, "role", foundPol.Targets[0].TargetType)
		assert.Equal(t, "editor", foundPol.Targets[0].TargetValue)

		// cleanup: join rows first, then leaves, then root resource
		_ = provider.DeletePermissionScopesByPermissionID(ctx, permID)
		_ = provider.DeletePermissionPoliciesByPermissionID(ctx, permID)
		_ = provider.DeletePermission(ctx, permID)
		_ = provider.DeletePolicyTargetsByPolicyID(ctx, polID)
		_ = provider.DeletePolicy(ctx, polID)
		_ = provider.DeleteScope(ctx, scopeID)
		_ = provider.DeleteResource(ctx, resID)
	})
}

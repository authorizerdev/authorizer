package storage

import (
	"context"
	"net"
	"os"
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

var dbTypes = []string{
	constants.DbTypePostgres,
	constants.DbTypeMongoDB,
	constants.DbTypeArangoDB,
	constants.DbTypeScyllaDB,
	constants.DbTypeCouchbaseDB,
	constants.DbTypeDynamoDB,
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
		cfg.DatabaseURL = "http://0.0.0.0:8000"
	}

	return cfg
}

func TestStorageProvider(t *testing.T) {
	// Initialize logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	for _, dbType := range dbTypes {
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

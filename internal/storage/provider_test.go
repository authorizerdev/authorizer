package storage

import (
	"context"
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
		cfg.DatabaseURL = "postgres://postgres:postgres@localhost:5432/postgres"
	case constants.DbTypeMongoDB:
		cfg.DatabaseURL = "mongodb://localhost:27017"
	case constants.DbTypeArangoDB:
		cfg.DatabaseURL = "http://localhost:8529"
	case constants.DbTypeScyllaDB:
		cfg.DatabaseURL = "127.0.0.1:9042"
	case constants.DbTypeCouchbaseDB:
		cfg.DatabaseURL = "couchbase://127.0.0.1"
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
			ctx := context.Background()
			provider, err := New(cfg, &Dependencies{
				Log: &logger,
			})
			if dbType != constants.DbTypeCouchbaseDB {
				require.NoError(t, err)
			}
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
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.Equal(t, user.Email, createdUser.Email)

	// Test GetUserByEmail
	fetchedUser, err := provider.GetUserByEmail(ctx, *user.Email)
	assert.NoError(t, err)
	assert.Equal(t, user.Email, fetchedUser.Email)

	// Test UpdateUser
	fetchedUser.GivenName = refs.NewStringRef("Updated")
	updatedUser, err := provider.UpdateUser(ctx, fetchedUser)
	assert.NoError(t, err)
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
	user.PhoneNumber = refs.NewStringRef("+1234567890")
	createdUser, err = provider.AddUser(ctx, user)
	assert.NoError(t, err)
	assert.NotNil(t, createdUser)
	assert.Equal(t, user.PhoneNumber, createdUser.PhoneNumber)

	// Test GetUserByID
	fetchedUser, err = provider.GetUserByID(ctx, createdUser.ID)
	assert.NoError(t, err)
	assert.NotNil(t, fetchedUser)
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
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "+3216549870", *user.PhoneNumber)

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
		EventName: "test_event",
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
	assert.Equal(t, template.EventName, templates[0].EventName)
	assert.Equal(t, template.Template, templates[0].Template)

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

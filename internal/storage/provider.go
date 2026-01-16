package storage

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/db/arangodb"
	"github.com/authorizerdev/authorizer/internal/storage/db/cassandradb"
	"github.com/authorizerdev/authorizer/internal/storage/db/couchbase"
	"github.com/authorizerdev/authorizer/internal/storage/db/dynamodb"
	"github.com/authorizerdev/authorizer/internal/storage/db/mongodb"
	"github.com/authorizerdev/authorizer/internal/storage/db/sql"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// Dependencies struct the data store provider
type Dependencies struct {
	Log *zerolog.Logger
}

// Provider is the interface which defines the methods for the database provider
type Provider interface {
	// AddUser to save user information in database
	AddUser(ctx context.Context, user *schemas.User) (*schemas.User, error)
	// UpdateUser to update user information in database
	UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error)
	// DeleteUser to delete user information from database
	DeleteUser(ctx context.Context, user *schemas.User) error
	// ListUsers to get list of users from database
	ListUsers(ctx context.Context, pagination *model.Pagination) ([]*schemas.User, *model.Pagination, error)
	// GetUserByEmail to get user information from database using email address
	GetUserByEmail(ctx context.Context, email string) (*schemas.User, error)
	// GetUserByPhoneNumber to get user information from database using phone number
	GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.User, error)
	// GetUserByID to get user information from database using user ID
	GetUserByID(ctx context.Context, id string) (*schemas.User, error)
	// UpdateUsers to update multiple users, with parameters of user IDs slice
	// If ids set to nil / empty all the users will be updated
	UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error

	// AddVerificationRequest to save verification request in database
	AddVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) (*schemas.VerificationRequest, error)
	// GetVerificationRequestByToken to get verification request from database using token
	GetVerificationRequestByToken(ctx context.Context, token string) (*schemas.VerificationRequest, error)
	// GetVerificationRequestByEmail to get verification request by email from database
	GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*schemas.VerificationRequest, error)
	// ListVerificationRequests to get list of verification requests from database
	ListVerificationRequests(ctx context.Context, pagination *model.Pagination) ([]*schemas.VerificationRequest, *model.Pagination, error)
	// DeleteVerificationRequest to delete verification request from database
	DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error

	// AddSession to save session information in database
	AddSession(ctx context.Context, session *schemas.Session) error
	// DeleteSession to delete session information from database
	DeleteSession(ctx context.Context, userId string) error

	// // AddEnv to save environment information in database
	// AddEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error)
	// // UpdateEnv to update environment information in database
	// UpdateEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error)
	// // GetEnv to get environment information from database
	// GetEnv(ctx context.Context) (*schemas.Env, error)

	// AddWebhook to add webhook
	AddWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error)
	// UpdateWebhook to update webhook
	UpdateWebhook(ctx context.Context, webhook *schemas.Webhook) (*schemas.Webhook, error)
	// ListWebhook to list webhook
	ListWebhook(ctx context.Context, pagination *model.Pagination) ([]*schemas.Webhook, *model.Pagination, error)
	// GetWebhookByID to get webhook by id
	GetWebhookByID(ctx context.Context, webhookID string) (*schemas.Webhook, error)
	// GetWebhookByEventName to get webhook by event_name
	GetWebhookByEventName(ctx context.Context, eventName string) ([]*schemas.Webhook, error)
	// DeleteWebhook to delete webhook
	DeleteWebhook(ctx context.Context, webhook *schemas.Webhook) error

	// AddWebhookLog to add webhook log
	AddWebhookLog(ctx context.Context, webhookLog *schemas.WebhookLog) (*schemas.WebhookLog, error)
	// ListWebhookLogs to list webhook logs
	ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) ([]*schemas.WebhookLog, *model.Pagination, error)

	// AddEmailTemplate to add EmailTemplate
	AddEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error)
	// UpdateEmailTemplate to update EmailTemplate
	UpdateEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*schemas.EmailTemplate, error)
	// ListEmailTemplate to list EmailTemplate
	ListEmailTemplate(ctx context.Context, pagination *model.Pagination) ([]*schemas.EmailTemplate, *model.Pagination, error)
	// GetEmailTemplateByID to get EmailTemplate by id
	GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*schemas.EmailTemplate, error)
	// GetEmailTemplateByEventName to get EmailTemplate by event_name
	GetEmailTemplateByEventName(ctx context.Context, eventName string) (*schemas.EmailTemplate, error)
	// DeleteEmailTemplate to delete EmailTemplate
	DeleteEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) error

	// UpsertOTP to add or update otp
	UpsertOTP(ctx context.Context, otp *schemas.OTP) (*schemas.OTP, error)
	// GetOTPByEmail to get otp for a given email address
	GetOTPByEmail(ctx context.Context, emailAddress string) (*schemas.OTP, error)
	// GetOTPByPhoneNumber to get otp for a given phone number
	GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*schemas.OTP, error)
	// DeleteOTP to delete otp
	DeleteOTP(ctx context.Context, otp *schemas.OTP) error

	// AddAuthenticator adds a new authenticator document to the database.
	// If the authenticator doesn't have an ID, a new one is generated.
	// The created document is returned, or an error if the operation fails.
	AddAuthenticator(ctx context.Context, totp *schemas.Authenticator) (*schemas.Authenticator, error)
	// UpdateAuthenticator updates an existing authenticator document in the database.
	// The updated document is returned, or an error if the operation fails.
	UpdateAuthenticator(ctx context.Context, totp *schemas.Authenticator) (*schemas.Authenticator, error)
	// GetAuthenticatorDetailsByUserId retrieves details of an authenticator document based on user ID and authenticator type.
	// If found, the authenticator document is returned, or an error if not found or an error occurs during the retrieval.
	GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*schemas.Authenticator, error)

	// Session Token methods (for database-backed memory store)
	// AddSessionToken adds a session token to the database
	AddSessionToken(ctx context.Context, token *schemas.SessionToken) error
	// GetSessionTokenByUserIDAndKey retrieves a session token by user ID and key
	GetSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.SessionToken, error)
	// DeleteSessionToken deletes a session token by ID
	DeleteSessionToken(ctx context.Context, id string) error
	// DeleteSessionTokenByUserIDAndKey deletes a session token by user ID and key
	DeleteSessionTokenByUserIDAndKey(ctx context.Context, userId, key string) error
	// DeleteAllSessionTokensByUserID deletes all session tokens for a user ID
	DeleteAllSessionTokensByUserID(ctx context.Context, userId string) error
	// DeleteSessionTokensByNamespace deletes all session tokens for a namespace (e.g., "auth_provider")
	DeleteSessionTokensByNamespace(ctx context.Context, namespace string) error
	// CleanExpiredSessionTokens removes expired session tokens from the database
	CleanExpiredSessionTokens(ctx context.Context) error
	// GetAllSessionTokens retrieves all session tokens (for testing)
	GetAllSessionTokens(ctx context.Context) ([]*schemas.SessionToken, error)

	// MFA Session methods (for database-backed memory store)
	// AddMFASession adds an MFA session to the database
	AddMFASession(ctx context.Context, session *schemas.MFASession) error
	// GetMFASessionByUserIDAndKey retrieves an MFA session by user ID and key
	GetMFASessionByUserIDAndKey(ctx context.Context, userId, key string) (*schemas.MFASession, error)
	// DeleteMFASession deletes an MFA session by ID
	DeleteMFASession(ctx context.Context, id string) error
	// DeleteMFASessionByUserIDAndKey deletes an MFA session by user ID and key
	DeleteMFASessionByUserIDAndKey(ctx context.Context, userId, key string) error
	// GetAllMFASessionsByUserID retrieves all MFA sessions for a user ID
	GetAllMFASessionsByUserID(ctx context.Context, userId string) ([]*schemas.MFASession, error)
	// CleanExpiredMFASessions removes expired MFA sessions from the database
	CleanExpiredMFASessions(ctx context.Context) error
	// GetAllMFASessions retrieves all MFA sessions (for testing)
	GetAllMFASessions(ctx context.Context) ([]*schemas.MFASession, error)

	// OAuth State methods (for database-backed memory store)
	// AddOAuthState adds an OAuth state to the database
	AddOAuthState(ctx context.Context, state *schemas.OAuthState) error
	// GetOAuthStateByKey retrieves an OAuth state by key
	GetOAuthStateByKey(ctx context.Context, key string) (*schemas.OAuthState, error)
	// DeleteOAuthStateByKey deletes an OAuth state by key
	DeleteOAuthStateByKey(ctx context.Context, key string) error
	// GetAllOAuthStates retrieves all OAuth states (for testing)
	GetAllOAuthStates(ctx context.Context) ([]*schemas.OAuthState, error)
}

// New creates a new database provider based on the configuration
func New(config *config.Config, deps *Dependencies) (Provider, error) {
	var provider Provider
	var err error
	if config.DatabaseType == "" {
		return nil, fmt.Errorf("database type is required")
	}

	switch config.DatabaseType {
	case constants.DbTypePostgres,
		constants.DbTypeSqlite,
		constants.DbTypeLibSQL,
		constants.DbTypeMysql,
		constants.DbTypeSqlserver,
		constants.DbTypeYugabyte,
		constants.DbTypeMariaDB,
		constants.DbTypeCockroachDB,
		constants.DbTypePlanetScaleDB:
		provider, err = sql.NewProvider(config, &sql.Dependencies{
			Log: deps.Log,
		})
	case constants.DbTypeMongoDB:
		provider, err = mongodb.NewProvider(config, &mongodb.Dependencies{
			Log: deps.Log,
		})
	case constants.DbTypeArangoDB:
		provider, err = arangodb.NewProvider(config, &arangodb.Dependencies{
			Log: deps.Log,
		})
	case constants.DbTypeCassandraDB,
		constants.DbTypeScyllaDB:
		provider, err = cassandradb.NewProvider(config, &cassandradb.Dependencies{
			Log: deps.Log,
		})
	case constants.DbTypeCouchbaseDB:
		provider, err = couchbase.NewProvider(config, &couchbase.Dependencies{
			Log: deps.Log,
		})
	case constants.DbTypeDynamoDB:
		provider, err = dynamodb.NewProvider(config, &dynamodb.Dependencies{
			Log: deps.Log,
		})
	default:
		err = fmt.Errorf("unsupported database type: %s", config.DatabaseType)

	}
	if err != nil {
		return nil, err
	}
	return provider, nil
}

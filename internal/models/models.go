package models

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/models/config"
	"github.com/authorizerdev/authorizer/internal/models/db/arangodb"
	"github.com/authorizerdev/authorizer/internal/models/db/cassandradb"
	"github.com/authorizerdev/authorizer/internal/models/db/couchbase"
	"github.com/authorizerdev/authorizer/internal/models/db/dynamodb"
	"github.com/authorizerdev/authorizer/internal/models/db/mongodb"
	"github.com/authorizerdev/authorizer/internal/models/db/sql"
	"github.com/authorizerdev/authorizer/internal/models/schemas"
)

// Provider is the interface which defines the methods for the database provider
type Provider interface {
	// AddUser to save user information in database
	AddUser(ctx context.Context, user *schemas.User) (*schemas.User, error)
	// UpdateUser to update user information in database
	UpdateUser(ctx context.Context, user *schemas.User) (*schemas.User, error)
	// DeleteUser to delete user information from database
	DeleteUser(ctx context.Context, user *schemas.User) error
	// ListUsers to get list of users from database
	ListUsers(ctx context.Context, pagination *model.Pagination) (*model.Users, error)
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
	ListVerificationRequests(ctx context.Context, pagination *model.Pagination) (*model.VerificationRequests, error)
	// DeleteVerificationRequest to delete verification request from database
	DeleteVerificationRequest(ctx context.Context, verificationRequest *schemas.VerificationRequest) error

	// AddSession to save session information in database
	AddSession(ctx context.Context, session *schemas.Session) error
	// DeleteSession to delete session information from database
	DeleteSession(ctx context.Context, userId string) error

	// AddEnv to save environment information in database
	AddEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error)
	// UpdateEnv to update environment information in database
	UpdateEnv(ctx context.Context, env *schemas.Env) (*schemas.Env, error)
	// GetEnv to get environment information from database
	GetEnv(ctx context.Context) (*schemas.Env, error)

	// AddWebhook to add webhook
	AddWebhook(ctx context.Context, webhook *schemas.Webhook) (*model.Webhook, error)
	// UpdateWebhook to update webhook
	UpdateWebhook(ctx context.Context, webhook *schemas.Webhook) (*model.Webhook, error)
	// ListWebhook to list webhook
	ListWebhook(ctx context.Context, pagination *model.Pagination) (*model.Webhooks, error)
	// GetWebhookByID to get webhook by id
	GetWebhookByID(ctx context.Context, webhookID string) (*model.Webhook, error)
	// GetWebhookByEventName to get webhook by event_name
	GetWebhookByEventName(ctx context.Context, eventName string) ([]*model.Webhook, error)
	// DeleteWebhook to delete webhook
	DeleteWebhook(ctx context.Context, webhook *model.Webhook) error

	// AddWebhookLog to add webhook log
	AddWebhookLog(ctx context.Context, webhookLog *schemas.WebhookLog) (*model.WebhookLog, error)
	// ListWebhookLogs to list webhook logs
	ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) (*model.WebhookLogs, error)

	// AddEmailTemplate to add EmailTemplate
	AddEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*model.EmailTemplate, error)
	// UpdateEmailTemplate to update EmailTemplate
	UpdateEmailTemplate(ctx context.Context, emailTemplate *schemas.EmailTemplate) (*model.EmailTemplate, error)
	// ListEmailTemplate to list EmailTemplate
	ListEmailTemplate(ctx context.Context, pagination *model.Pagination) (*model.EmailTemplates, error)
	// GetEmailTemplateByID to get EmailTemplate by id
	GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*model.EmailTemplate, error)
	// GetEmailTemplateByEventName to get EmailTemplate by event_name
	GetEmailTemplateByEventName(ctx context.Context, eventName string) (*model.EmailTemplate, error)
	// DeleteEmailTemplate to delete EmailTemplate
	DeleteEmailTemplate(ctx context.Context, emailTemplate *model.EmailTemplate) error

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
}

// New creates a new database provider based on the configuration
func New(config config.Config, deps config.Dependencies) (Provider, error) {
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
		provider, err = sql.NewProvider(config, deps)
	case constants.DbTypeMongoDB:
		provider, err = mongodb.NewProvider(config, deps)
	case constants.DbTypeArangoDB:
		provider, err = arangodb.NewProvider(config, deps)
	case constants.DbTypeCassandraDB,
		constants.DbTypeScyllaDB:
		provider, err = cassandradb.NewProvider(config, deps)
	case constants.DbTypeCouchbaseDB:
		provider, err = couchbase.NewProvider(config, deps)
	case constants.DbTypeDynamoDB:
		provider, err = dynamodb.NewProvider(config, deps)
	default:
		err = fmt.Errorf("unsupported database type: %s", config.DatabaseType)

	}
	if err != nil {
		return nil, err
	}
	return provider, nil
}

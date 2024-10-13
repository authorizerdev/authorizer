package models

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/db/models"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/models/db/sql"
)

// Config is the configuration for the database
type Config struct {
	// DatabaseType is the type of database to use
	DatabaseType string
	// DatabaseURL is the URL of the database
	DatabaseURL string
}

// Dependencies for the database
type Dependencies struct {
	Log zerolog.Logger
}

// Provider is the interface which defines the methods for the database provider
type Provider interface {
	// AddUser to save user information in database
	AddUser(ctx context.Context, user *models.User) (*models.User, error)
	// UpdateUser to update user information in database
	UpdateUser(ctx context.Context, user *models.User) (*models.User, error)
	// DeleteUser to delete user information from database
	DeleteUser(ctx context.Context, user *models.User) error
	// ListUsers to get list of users from database
	ListUsers(ctx context.Context, pagination *model.Pagination) (*model.Users, error)
	// GetUserByEmail to get user information from database using email address
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	// GetUserByPhoneNumber to get user information from database using phone number
	GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*models.User, error)
	// GetUserByID to get user information from database using user ID
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	// UpdateUsers to update multiple users, with parameters of user IDs slice
	// If ids set to nil / empty all the users will be updated
	UpdateUsers(ctx context.Context, data map[string]interface{}, ids []string) error

	// AddVerificationRequest to save verification request in database
	AddVerificationRequest(ctx context.Context, verificationRequest *models.VerificationRequest) (*models.VerificationRequest, error)
	// GetVerificationRequestByToken to get verification request from database using token
	GetVerificationRequestByToken(ctx context.Context, token string) (*models.VerificationRequest, error)
	// GetVerificationRequestByEmail to get verification request by email from database
	GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (*models.VerificationRequest, error)
	// ListVerificationRequests to get list of verification requests from database
	ListVerificationRequests(ctx context.Context, pagination *model.Pagination) (*model.VerificationRequests, error)
	// DeleteVerificationRequest to delete verification request from database
	DeleteVerificationRequest(ctx context.Context, verificationRequest *models.VerificationRequest) error

	// AddSession to save session information in database
	AddSession(ctx context.Context, session *models.Session) error
	// DeleteSession to delete session information from database
	DeleteSession(ctx context.Context, userId string) error

	// AddEnv to save environment information in database
	AddEnv(ctx context.Context, env *models.Env) (*models.Env, error)
	// UpdateEnv to update environment information in database
	UpdateEnv(ctx context.Context, env *models.Env) (*models.Env, error)
	// GetEnv to get environment information from database
	GetEnv(ctx context.Context) (*models.Env, error)

	// AddWebhook to add webhook
	AddWebhook(ctx context.Context, webhook *models.Webhook) (*model.Webhook, error)
	// UpdateWebhook to update webhook
	UpdateWebhook(ctx context.Context, webhook *models.Webhook) (*model.Webhook, error)
	// ListWebhook to list webhook
	ListWebhook(ctx context.Context, pagination *model.Pagination) (*model.Webhooks, error)
	// GetWebhookByID to get webhook by id
	GetWebhookByID(ctx context.Context, webhookID string) (*model.Webhook, error)
	// GetWebhookByEventName to get webhook by event_name
	GetWebhookByEventName(ctx context.Context, eventName string) ([]*model.Webhook, error)
	// DeleteWebhook to delete webhook
	DeleteWebhook(ctx context.Context, webhook *model.Webhook) error

	// AddWebhookLog to add webhook log
	AddWebhookLog(ctx context.Context, webhookLog *models.WebhookLog) (*model.WebhookLog, error)
	// ListWebhookLogs to list webhook logs
	ListWebhookLogs(ctx context.Context, pagination *model.Pagination, webhookID string) (*model.WebhookLogs, error)

	// AddEmailTemplate to add EmailTemplate
	AddEmailTemplate(ctx context.Context, emailTemplate *models.EmailTemplate) (*model.EmailTemplate, error)
	// UpdateEmailTemplate to update EmailTemplate
	UpdateEmailTemplate(ctx context.Context, emailTemplate *models.EmailTemplate) (*model.EmailTemplate, error)
	// ListEmailTemplate to list EmailTemplate
	ListEmailTemplate(ctx context.Context, pagination *model.Pagination) (*model.EmailTemplates, error)
	// GetEmailTemplateByID to get EmailTemplate by id
	GetEmailTemplateByID(ctx context.Context, emailTemplateID string) (*model.EmailTemplate, error)
	// GetEmailTemplateByEventName to get EmailTemplate by event_name
	GetEmailTemplateByEventName(ctx context.Context, eventName string) (*model.EmailTemplate, error)
	// DeleteEmailTemplate to delete EmailTemplate
	DeleteEmailTemplate(ctx context.Context, emailTemplate *model.EmailTemplate) error

	// UpsertOTP to add or update otp
	UpsertOTP(ctx context.Context, otp *models.OTP) (*models.OTP, error)
	// GetOTPByEmail to get otp for a given email address
	GetOTPByEmail(ctx context.Context, emailAddress string) (*models.OTP, error)
	// GetOTPByPhoneNumber to get otp for a given phone number
	GetOTPByPhoneNumber(ctx context.Context, phoneNumber string) (*models.OTP, error)
	// DeleteOTP to delete otp
	DeleteOTP(ctx context.Context, otp *models.OTP) error

	// AddAuthenticator adds a new authenticator document to the database.
	// If the authenticator doesn't have an ID, a new one is generated.
	// The created document is returned, or an error if the operation fails.
	AddAuthenticator(ctx context.Context, totp *models.Authenticator) (*models.Authenticator, error)
	// UpdateAuthenticator updates an existing authenticator document in the database.
	// The updated document is returned, or an error if the operation fails.
	UpdateAuthenticator(ctx context.Context, totp *models.Authenticator) (*models.Authenticator, error)
	// GetAuthenticatorDetailsByUserId retrieves details of an authenticator document based on user ID and authenticator type.
	// If found, the authenticator document is returned, or an error if not found or an error occurs during the retrieval.
	GetAuthenticatorDetailsByUserId(ctx context.Context, userId string, authenticatorType string) (*models.Authenticator, error)
}

// New creates a new database provider based on the configuration
func New(config Config, deps Dependencies) (Provider, error) {
	var provider Provider
	var err error
	switch config.DatabaseType {
	case constants.DbTypePostgres,
		constants.DbTypeSqlite,
		constants.DbTypeMysql,
		constants.DbTypeSqlserver,
		constants.DbTypeYugabyte,
		constants.DbTypeMariaDB,
		constants.DbTypePlanetScaleDB:
		provider, err = sql.NewProvider(config, deps)
	default:
		err = fmt.Errorf("unsupported database type: %s", config.DatabaseType)

	}
	if err != nil {
		return nil, err
	}
	return provider, nil
}

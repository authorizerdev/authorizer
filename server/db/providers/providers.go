package providers

import (
	"context"

	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
)

type Provider interface {
	// AddUser to save user information in database
	AddUser(ctx context.Context, user models.User) (models.User, error)
	// UpdateUser to update user information in database
	UpdateUser(ctx context.Context, user models.User) (models.User, error)
	// DeleteUser to delete user information from database
	DeleteUser(ctx context.Context, user models.User) error
	// ListUsers to get list of users from database
	ListUsers(ctx context.Context, pagination model.Pagination) (*model.Users, error)
	// GetUserByEmail to get user information from database using email address
	GetUserByEmail(ctx context.Context, email string) (models.User, error)
	// GetUserByID to get user information from database using user ID
	GetUserByID(ctx context.Context, id string) (models.User, error)

	// AddVerification to save verification request in database
	AddVerificationRequest(ctx context.Context, verificationRequest models.VerificationRequest) (models.VerificationRequest, error)
	// GetVerificationRequestByToken to get verification request from database using token
	GetVerificationRequestByToken(ctx context.Context, token string) (models.VerificationRequest, error)
	// GetVerificationRequestByEmail to get verification request by email from database
	GetVerificationRequestByEmail(ctx context.Context, email string, identifier string) (models.VerificationRequest, error)
	// ListVerificationRequests to get list of verification requests from database
	ListVerificationRequests(ctx context.Context, pagination model.Pagination) (*model.VerificationRequests, error)
	// DeleteVerificationRequest to delete verification request from database
	DeleteVerificationRequest(ctx context.Context, verificationRequest models.VerificationRequest) error

	// AddSession to save session information in database
	AddSession(ctx context.Context, session models.Session) error

	// AddEnv to save environment information in database
	AddEnv(ctx context.Context, env models.Env) (models.Env, error)
	// UpdateEnv to update environment information in database
	UpdateEnv(ctx context.Context, env models.Env) (models.Env, error)
	// GetEnv to get environment information from database
	GetEnv(ctx context.Context) (models.Env, error)

	// AddWebhook to add webhook
	AddWebhook(ctx context.Context, webhook models.Webhook) (*model.Webhook, error)
	// UpdateWebhook to update webhook
	UpdateWebhook(ctx context.Context, webhook models.Webhook) (*model.Webhook, error)
	// ListWebhooks to list webhook
	ListWebhook(ctx context.Context, pagination model.Pagination) (*model.Webhooks, error)
	// GetWebhookByID to get webhook by id
	GetWebhookByID(ctx context.Context, webhookID string) (*model.Webhook, error)
	// GetWebhookByEventName to get webhook by event_name
	GetWebhookByEventName(ctx context.Context, eventName string) (*model.Webhook, error)
	// DeleteWebhook to delete webhook
	DeleteWebhook(ctx context.Context, webhook *model.Webhook) error

	// AddWebhookLog to add webhook log
	AddWebhookLog(ctx context.Context, webhookLog models.WebhookLog) (*model.WebhookLog, error)
	// ListWebhookLogs to list webhook logs
	ListWebhookLogs(ctx context.Context, pagination model.Pagination, webhookID string) (*model.WebhookLogs, error)
}

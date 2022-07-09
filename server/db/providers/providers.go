package providers

import (
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
)

type Provider interface {
	// AddUser to save user information in database
	AddUser(user models.User) (models.User, error)
	// UpdateUser to update user information in database
	UpdateUser(user models.User) (models.User, error)
	// DeleteUser to delete user information from database
	DeleteUser(user models.User) error
	// ListUsers to get list of users from database
	ListUsers(pagination model.Pagination) (*model.Users, error)
	// GetUserByEmail to get user information from database using email address
	GetUserByEmail(email string) (models.User, error)
	// GetUserByID to get user information from database using user ID
	GetUserByID(id string) (models.User, error)

	// AddVerification to save verification request in database
	AddVerificationRequest(verificationRequest models.VerificationRequest) (models.VerificationRequest, error)
	// GetVerificationRequestByToken to get verification request from database using token
	GetVerificationRequestByToken(token string) (models.VerificationRequest, error)
	// GetVerificationRequestByEmail to get verification request by email from database
	GetVerificationRequestByEmail(email string, identifier string) (models.VerificationRequest, error)
	// ListVerificationRequests to get list of verification requests from database
	ListVerificationRequests(pagination model.Pagination) (*model.VerificationRequests, error)
	// DeleteVerificationRequest to delete verification request from database
	DeleteVerificationRequest(verificationRequest models.VerificationRequest) error

	// AddSession to save session information in database
	AddSession(session models.Session) error
	// DeleteSession to delete session information from database
	DeleteSession(userId string) error

	// AddEnv to save environment information in database
	AddEnv(env models.Env) (models.Env, error)
	// UpdateEnv to update environment information in database
	UpdateEnv(env models.Env) (models.Env, error)
	// GetEnv to get environment information from database
	GetEnv() (models.Env, error)

	// AddWebhook to add webhook
	AddWebhook(webhook models.Webhook) (models.Webhook, error)
	// UpdateWebhook to update webhook
	UpdateWebhook(webhook models.Webhook) (models.Webhook, error)
	// ListWebhooks to list webhook
	ListWebhook(pagination model.Pagination) (*model.Webhooks, error)
	// GetWebhookByID to get webhook by id
	GetWebhookByID(webhookID string) (models.Webhook, error)
	// GetWebhookByEventName to get webhook by event_name
	GetWebhookByEventName(eventName string) (models.Webhook, error)
	// DeleteWebhook to delete webhook
	DeleteWebhook(webhookID string) error

	// AddWebhookLog to add webhook log
	AddWebhookLog(webhookLog models.WebhookLog) (models.WebhookLog, error)
	// ListWebhookLogs to list webhook logs
	ListWebhookLogs(pagination model.Pagination, webhookID string) (*model.WebhookLogs, error)
}

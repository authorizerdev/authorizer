package graphql

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/authenticators"
	"github.com/authorizerdev/authorizer/internal/authorization/engine"
	"github.com/authorizerdev/authorizer/internal/config"
	"github.com/authorizerdev/authorizer/internal/email"
	"github.com/authorizerdev/authorizer/internal/events"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/memory_store"
	"github.com/authorizerdev/authorizer/internal/service"
	"github.com/authorizerdev/authorizer/internal/sms"
	"github.com/authorizerdev/authorizer/internal/storage"
	"github.com/authorizerdev/authorizer/internal/token"
)

// Dependencies for a graphql provider
type Dependencies struct {
	Log *zerolog.Logger

	// Providers for various services
	// AuditProvider is used to log audit events
	AuditProvider audit.Provider
	// AuthenticatorProvider is used to register authenticators like totp (Google Authenticator)
	AuthenticatorProvider authenticators.Provider
	// EmailProvider is used to send emails
	EmailProvider email.Provider
	// EventsProvider is used to register events
	EventsProvider events.Provider
	// MemoryStoreProvider is used to store data in memory
	MemoryStoreProvider memory_store.Provider
	// SMSProvider is used to send SMS
	SMSProvider sms.Provider
	// StorageProvider is used to register storage like database
	StorageProvider storage.Provider
	// TokenProvider is used to generate tokens
	TokenProvider token.Provider
	// ServiceProvider hosts the transport-agnostic public-API operations.
	// Resolvers for migrated ops delegate here.
	ServiceProvider service.Provider
	// AuthzEngine is the fine-grained authorization (FGA) engine.
	// It is nil unless an FGA store is configured (--fga-store);
	// resolvers MUST fail closed (return an error) when it is nil.
	AuthzEngine engine.AuthorizationEngine
}

// New constructs a new graphql provider with given arguments
func New(cfg *config.Config, deps *Dependencies) (Provider, error) {
	// TODO - Add any validation here for config and dependencies
	g := &graphqlProvider{
		Config:       cfg,
		Dependencies: *deps,
	}
	return g, nil
}

// graphqlProvider is the struct that provides resolver functions.
type graphqlProvider struct {
	*config.Config
	Dependencies
}

// Ensure that graphqlProvider implements the Provider interface
var _ Provider = &graphqlProvider{}

// adminService returns the admin operations of the underlying service provider.
// The concrete service value implements both service.Provider and
// service.AdminProvider (compile-time asserted in the service package), so this
// assertion always succeeds; admin resolvers delegate through it.
func (g *graphqlProvider) adminService() service.AdminProvider {
	return g.ServiceProvider.(service.AdminProvider)
}

// Provider is the interface that provides the methods to interact with the graphql mutations and queries.
type Provider interface {
	// AddEmailTemplate is the method to add email template.
	// Permissions: authorizer:admin
	AddEmailTemplate(ctx context.Context, params *model.AddEmailTemplateRequest) (*model.Response, error)
	// AddWebhook is the method to add webhook.
	// Permissions: authorizer:admin
	AddWebhook(ctx context.Context, params *model.AddWebhookRequest) (*model.Response, error)
	// AdminLogin is the method to login as admin.
	// Permissions: none
	AdminLogin(ctx context.Context, params *model.AdminLoginRequest) (*model.Response, error)
	// AdminLogout is the method to logout as admin.
	// Permissions: authorizer:admin
	AdminLogout(ctx context.Context) (*model.Response, error)
	// AdminSession is the method to get admin session.
	// Permissions: authorizer:admin
	AdminSession(ctx context.Context) (*model.Response, error)
	// DeactivateAccount is the method to deactivate account.
	// Permissions: authorized user
	DeactivateAccount(ctx context.Context) (*model.Response, error)
	// DeleteEmailTemplate is the method to delete email template.
	// Permissions: authorizer:admin
	DeleteEmailTemplate(ctx context.Context, params *model.DeleteEmailTemplateRequest) (*model.Response, error)
	// DeleteUser is the method to delete user.
	// Permissions: authorizer:admin
	DeleteUser(ctx context.Context, params *model.DeleteUserRequest) (*model.Response, error)
	// DeleteWebhook is the method to delete webhook.
	// Permissions: authorizer:admin
	DeleteWebhook(ctx context.Context, params *model.WebhookRequest) (*model.Response, error)
	// EmailTemplates is the method to list email templates.
	// Permissions: authorizer:admin
	EmailTemplates(ctx context.Context, in *model.PaginatedRequest) (*model.EmailTemplates, error)
	// EnableAccess is the method to enable access.
	// Permissions: authorizer:admin
	EnableAccess(ctx context.Context, params *model.UpdateAccessRequest) (*model.Response, error)
	// ForgotPassword is the method to forgot password.
	// Permissions: none
	ForgotPassword(ctx context.Context, params *model.ForgotPasswordRequest) (*model.ForgotPasswordResponse, error)
	// InviteMembers is the method to invite members.
	// Permissions: authorizer:admin
	InviteMembers(ctx context.Context, params *model.InviteMemberRequest) (*model.InviteMembersResponse, error)
	// Login is the method to login.
	// Permissions: none
	Login(ctx context.Context, params *model.LoginRequest) (*model.AuthResponse, error)
	// Logout is the method to logout.
	// Permissions: authorized user
	Logout(ctx context.Context) (*model.Response, error)
	// MagicLinkLogin is the method to login using magic link.
	// Permissions: none
	MagicLinkLogin(ctx context.Context, params *model.MagicLinkLoginRequest) (*model.Response, error)
	// Meta is the method to get meta.
	// Permissions: none
	Meta(ctx context.Context) (*model.Meta, error)
	// AdminMeta returns admin-only configuration metadata (e.g. the configured
	// roles), the non-deprecated replacement for the bits of _env the dashboard
	// needs.
	// Permissions: authorizer:admin
	AdminMeta(ctx context.Context) (*model.AdminMeta, error)
	// Profile is the method to get profile.
	// Permissions: authorized user
	Profile(ctx context.Context) (*model.User, error)
	// ResendOTP is the method to resend OTP.
	// Permissions: none
	ResendOTP(ctx context.Context, params *model.ResendOTPRequest) (*model.Response, error)
	// ResendVerifyEmail is the method to resend verification email.
	// Permissions: none
	ResendVerifyEmail(ctx context.Context, params *model.ResendVerifyEmailRequest) (*model.Response, error)
	// ResetPassword is the method to reset password.
	// Permissions: none
	ResetPassword(ctx context.Context, params *model.ResetPasswordRequest) (*model.Response, error)
	// RevokeAccess is the method to revoke access.
	// Permissions: authorizer:admin
	RevokeAccess(ctx context.Context, params *model.UpdateAccessRequest) (*model.Response, error)
	// Revoke is the method to revoke refresh token.
	// Permissions: none
	Revoke(ctx context.Context, params *model.OAuthRevokeRequest) (*model.Response, error)
	// Session is the method to get session.
	// Permissions: authorized user
	Session(ctx context.Context, params *model.SessionQueryRequest) (*model.AuthResponse, error)
	// SignUp is the method to SignUp.
	// Permissions: none
	SignUp(ctx context.Context, params *model.SignUpRequest) (*model.AuthResponse, error)
	// TestEndpoint is the method to test endpoint.
	// Permissions: authorizer:admin
	TestEndpoint(ctx context.Context, params *model.TestEndpointRequest) (*model.TestEndpointResponse, error)
	// UpdateEmailTemplate is the method to update email template.
	// Permissions: authorizer:admin
	UpdateEmailTemplate(ctx context.Context, params *model.UpdateEmailTemplateRequest) (*model.Response, error)
	// UpdateProfile is the method to update profile.
	// Permissions: authorized user
	UpdateProfile(ctx context.Context, params *model.UpdateProfileRequest) (*model.Response, error)
	// UpdateUser is the method to update user.
	// Permissions: authorizer:admin
	UpdateUser(ctx context.Context, params *model.UpdateUserRequest) (*model.User, error)
	// UpdateWebhook is the method to update webhook.
	// Permissions: authorizer:admin
	UpdateWebhook(ctx context.Context, params *model.UpdateWebhookRequest) (*model.Response, error)
	// User is the method to get user.
	// Permissions: authorizer:admin
	User(ctx context.Context, params *model.GetUserRequest) (*model.User, error)
	// Users is the method to list users.
	// Permissions: authorizer:admin
	Users(ctx context.Context, in *model.PaginatedRequest) (*model.Users, error)
	// ValidateJWTToken is the method to validate JWT token.
	// Permissions: none
	ValidateJWTToken(ctx context.Context, params *model.ValidateJWTTokenRequest) (*model.ValidateJWTTokenResponse, error)
	// ValidateSession is the method to validate browser session.
	// Permissions: authorized user
	ValidateSession(ctx context.Context, params *model.ValidateSessionRequest) (*model.ValidateSessionResponse, error)
	// VerificationRequests is the method to list verification requests.
	// Permissions: authorizer:admin
	VerificationRequests(ctx context.Context, in *model.PaginatedRequest) (*model.VerificationRequests, error)
	// VerifyEmail is the method to verify email.
	// Permissions: none
	VerifyEmail(ctx context.Context, params *model.VerifyEmailRequest) (*model.AuthResponse, error)
	// VerifyOTP is the method to verify OTP.
	// Permissions: authorized otp request
	VerifyOTP(ctx context.Context, params *model.VerifyOTPRequest) (*model.AuthResponse, error)
	// AuditLogs is the method to list audit logs.
	// Permissions: authorizer:admin
	AuditLogs(ctx context.Context, params *model.ListAuditLogRequest) (*model.AuditLogs, error)
	// WebhookLogs is the method to list webhook logs.
	// Permissions: authorizer:admin
	WebhookLogs(ctx context.Context, in *model.ListWebhookLogRequest) (*model.WebhookLogs, error)
	// Webhook is the method to get webhook.
	// Permissions: authorizer:admin
	Webhook(ctx context.Context, params *model.WebhookRequest) (*model.Webhook, error)
	// Webhooks is the method to list webhooks.
	// Permissions: authorizer:admin
	Webhooks(ctx context.Context, in *model.PaginatedRequest) (*model.Webhooks, error)
	// CreateClient creates a machine/workload service account.
	// Permissions: authorizer:admin
	CreateClient(ctx context.Context, params *model.CreateClientRequest) (*model.CreateClientResponse, error)
	// UpdateClient updates a service account.
	// Permissions: authorizer:admin
	UpdateClient(ctx context.Context, params *model.UpdateClientRequest) (*model.Client, error)
	// DeleteClient deletes a service account (cascades to trusted issuers).
	// Permissions: authorizer:admin
	DeleteClient(ctx context.Context, params *model.ClientRequest) (*model.Response, error)
	// RotateClientSecret rotates a service account's client secret.
	// Permissions: authorizer:admin
	RotateClientSecret(ctx context.Context, params *model.ClientRequest) (*model.CreateClientResponse, error)
	// Client returns a single service account by id.
	// Permissions: authorizer:admin
	Client(ctx context.Context, params *model.ClientRequest) (*model.Client, error)
	// Clients lists service accounts.
	// Permissions: authorizer:admin
	Clients(ctx context.Context, params *model.ListClientsRequest) (*model.Clients, error)
	// AddTrustedIssuer registers an external JWT issuer for a service account.
	// Permissions: authorizer:admin
	AddTrustedIssuer(ctx context.Context, params *model.AddTrustedIssuerRequest) (*model.TrustedIssuer, error)
	// UpdateTrustedIssuer updates a trusted issuer.
	// Permissions: authorizer:admin
	UpdateTrustedIssuer(ctx context.Context, params *model.UpdateTrustedIssuerRequest) (*model.TrustedIssuer, error)
	// DeleteTrustedIssuer deletes a trusted issuer.
	// Permissions: authorizer:admin
	DeleteTrustedIssuer(ctx context.Context, params *model.TrustedIssuerRequest) (*model.Response, error)
	// TrustedIssuer returns a single trusted issuer by id.
	// Permissions: authorizer:admin
	TrustedIssuer(ctx context.Context, params *model.TrustedIssuerRequest) (*model.TrustedIssuer, error)
	// TrustedIssuers lists trusted issuers, optionally filtered by service account.
	// Permissions: authorizer:admin
	TrustedIssuers(ctx context.Context, params *model.ListTrustedIssuersRequest) (*model.TrustedIssuers, error)
	// FgaWriteModel installs a new fine-grained authorization model.
	// Permissions: authorizer:admin
	FgaWriteModel(ctx context.Context, params *model.FgaWriteModelInput) (*model.FgaModel, error)
	// FgaGetModel returns the active fine-grained authorization model.
	// Permissions: authorizer:admin
	FgaGetModel(ctx context.Context) (*model.FgaModel, error)
	// FgaWriteTuples writes fine-grained authorization tuples.
	// Permissions: authorizer:admin
	FgaWriteTuples(ctx context.Context, params *model.FgaWriteTuplesInput) (*model.Response, error)
	// FgaDeleteTuples deletes fine-grained authorization tuples.
	// Permissions: authorizer:admin
	FgaDeleteTuples(ctx context.Context, params *model.FgaWriteTuplesInput) (*model.Response, error)
	// FgaReset deletes the entire authorization store (model, all versions and
	// tuples) and starts fresh. Refused while tuples still exist.
	// Permissions: authorizer:admin
	FgaReset(ctx context.Context) (*model.Response, error)
	// FgaReadTuples reads a page of fine-grained authorization tuples.
	// Permissions: authorizer:admin
	FgaReadTuples(ctx context.Context, params *model.FgaReadTuplesInput) (*model.FgaTuples, error)
	// FgaListUsers lists the users that have a relation on an object (reveals the
	// access graph).
	// Permissions: authorizer:admin
	FgaListUsers(ctx context.Context, params *model.FgaListUsersInput) (*model.FgaListUsersResponse, error)
	// FgaExpand returns the relationship/userset tree for a (relation, object).
	// Permissions: authorizer:admin
	FgaExpand(ctx context.Context, params *model.FgaExpandInput) (*model.FgaExpandResponse, error)
	// CheckPermissions evaluates one or more permission checks for the subject
	// (token-resolved by default; explicit user for super-admins or self).
	// Permissions: authorized user
	CheckPermissions(ctx context.Context, params *model.CheckPermissionsInput) (*model.CheckPermissionsResponse, error)
	// ListPermissions enumerates the objects the subject holds a permission on.
	// Permissions: authorized user
	ListPermissions(ctx context.Context, params *model.ListPermissionsInput) (*model.ListPermissionsResponse, error)
}

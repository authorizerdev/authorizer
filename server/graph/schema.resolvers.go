package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/server/graph/generated"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
)

// Signup is the resolver for the signup field.
func (r *mutationResolver) Signup(ctx context.Context, params model.SignUpInput) (*model.AuthResponse, error) {
	return resolvers.SignupResolver(ctx, params)
}

// MobileSignup is the resolver for the mobile_signup field.
func (r *mutationResolver) MobileSignup(ctx context.Context, params *model.MobileSignUpInput) (*model.AuthResponse, error) {
	return resolvers.MobileSignupResolver(ctx, params)
}

// Login is the resolver for the login field.
func (r *mutationResolver) Login(ctx context.Context, params model.LoginInput) (*model.AuthResponse, error) {
	return resolvers.LoginResolver(ctx, params)
}

// MobileLogin is the resolver for the mobile_login field.
func (r *mutationResolver) MobileLogin(ctx context.Context, params model.MobileLoginInput) (*model.AuthResponse, error) {
	return resolvers.MobileLoginResolver(ctx, params)
}

// MagicLinkLogin is the resolver for the magic_link_login field.
func (r *mutationResolver) MagicLinkLogin(ctx context.Context, params model.MagicLinkLoginInput) (*model.Response, error) {
	return resolvers.MagicLinkLoginResolver(ctx, params)
}

// Logout is the resolver for the logout field.
func (r *mutationResolver) Logout(ctx context.Context) (*model.Response, error) {
	return resolvers.LogoutResolver(ctx)
}

// UpdateProfile is the resolver for the update_profile field.
func (r *mutationResolver) UpdateProfile(ctx context.Context, params model.UpdateProfileInput) (*model.Response, error) {
	return resolvers.UpdateProfileResolver(ctx, params)
}

// VerifyEmail is the resolver for the verify_email field.
func (r *mutationResolver) VerifyEmail(ctx context.Context, params model.VerifyEmailInput) (*model.AuthResponse, error) {
	return resolvers.VerifyEmailResolver(ctx, params)
}

// ResendVerifyEmail is the resolver for the resend_verify_email field.
func (r *mutationResolver) ResendVerifyEmail(ctx context.Context, params model.ResendVerifyEmailInput) (*model.Response, error) {
	return resolvers.ResendVerifyEmailResolver(ctx, params)
}

// ForgotPassword is the resolver for the forgot_password field.
func (r *mutationResolver) ForgotPassword(ctx context.Context, params model.ForgotPasswordInput) (*model.Response, error) {
	return resolvers.ForgotPasswordResolver(ctx, params)
}

// ResetPassword is the resolver for the reset_password field.
func (r *mutationResolver) ResetPassword(ctx context.Context, params model.ResetPasswordInput) (*model.Response, error) {
	return resolvers.ResetPasswordResolver(ctx, params)
}

// Revoke is the resolver for the revoke field.
func (r *mutationResolver) Revoke(ctx context.Context, params model.OAuthRevokeInput) (*model.Response, error) {
	return resolvers.RevokeResolver(ctx, params)
}

// VerifyOtp is the resolver for the verify_otp field.
func (r *mutationResolver) VerifyOtp(ctx context.Context, params model.VerifyOTPRequest) (*model.AuthResponse, error) {
	return resolvers.VerifyOtpResolver(ctx, params)
}

// ResendOtp is the resolver for the resend_otp field.
func (r *mutationResolver) ResendOtp(ctx context.Context, params model.ResendOTPRequest) (*model.Response, error) {
	return resolvers.ResendOTPResolver(ctx, params)
}

// DeactivateAccount is the resolver for the deactivate_account field.
func (r *mutationResolver) DeactivateAccount(ctx context.Context) (*model.Response, error) {
	panic(fmt.Errorf("not implemented: DeactivateAccount - deactivate_account"))
}

// DeleteUser is the resolver for the _delete_user field.
func (r *mutationResolver) DeleteUser(ctx context.Context, params model.DeleteUserInput) (*model.Response, error) {
	return resolvers.DeleteUserResolver(ctx, params)
}

// UpdateUser is the resolver for the _update_user field.
func (r *mutationResolver) UpdateUser(ctx context.Context, params model.UpdateUserInput) (*model.User, error) {
	return resolvers.UpdateUserResolver(ctx, params)
}

// AdminSignup is the resolver for the _admin_signup field.
func (r *mutationResolver) AdminSignup(ctx context.Context, params model.AdminSignupInput) (*model.Response, error) {
	return resolvers.AdminSignupResolver(ctx, params)
}

// AdminLogin is the resolver for the _admin_login field.
func (r *mutationResolver) AdminLogin(ctx context.Context, params model.AdminLoginInput) (*model.Response, error) {
	return resolvers.AdminLoginResolver(ctx, params)
}

// AdminLogout is the resolver for the _admin_logout field.
func (r *mutationResolver) AdminLogout(ctx context.Context) (*model.Response, error) {
	return resolvers.AdminLogoutResolver(ctx)
}

// UpdateEnv is the resolver for the _update_env field.
func (r *mutationResolver) UpdateEnv(ctx context.Context, params model.UpdateEnvInput) (*model.Response, error) {
	return resolvers.UpdateEnvResolver(ctx, params)
}

// InviteMembers is the resolver for the _invite_members field.
func (r *mutationResolver) InviteMembers(ctx context.Context, params model.InviteMemberInput) (*model.InviteMembersResponse, error) {
	return resolvers.InviteMembersResolver(ctx, params)
}

// RevokeAccess is the resolver for the _revoke_access field.
func (r *mutationResolver) RevokeAccess(ctx context.Context, param model.UpdateAccessInput) (*model.Response, error) {
	return resolvers.RevokeAccessResolver(ctx, param)
}

// EnableAccess is the resolver for the _enable_access field.
func (r *mutationResolver) EnableAccess(ctx context.Context, param model.UpdateAccessInput) (*model.Response, error) {
	return resolvers.EnableAccessResolver(ctx, param)
}

// GenerateJwtKeys is the resolver for the _generate_jwt_keys field.
func (r *mutationResolver) GenerateJwtKeys(ctx context.Context, params model.GenerateJWTKeysInput) (*model.GenerateJWTKeysResponse, error) {
	return resolvers.GenerateJWTKeysResolver(ctx, params)
}

// AddWebhook is the resolver for the _add_webhook field.
func (r *mutationResolver) AddWebhook(ctx context.Context, params model.AddWebhookRequest) (*model.Response, error) {
	return resolvers.AddWebhookResolver(ctx, params)
}

// UpdateWebhook is the resolver for the _update_webhook field.
func (r *mutationResolver) UpdateWebhook(ctx context.Context, params model.UpdateWebhookRequest) (*model.Response, error) {
	return resolvers.UpdateWebhookResolver(ctx, params)
}

// DeleteWebhook is the resolver for the _delete_webhook field.
func (r *mutationResolver) DeleteWebhook(ctx context.Context, params model.WebhookRequest) (*model.Response, error) {
	return resolvers.DeleteWebhookResolver(ctx, params)
}

// TestEndpoint is the resolver for the _test_endpoint field.
func (r *mutationResolver) TestEndpoint(ctx context.Context, params model.TestEndpointRequest) (*model.TestEndpointResponse, error) {
	return resolvers.TestEndpointResolver(ctx, params)
}

// AddEmailTemplate is the resolver for the _add_email_template field.
func (r *mutationResolver) AddEmailTemplate(ctx context.Context, params model.AddEmailTemplateRequest) (*model.Response, error) {
	return resolvers.AddEmailTemplateResolver(ctx, params)
}

// UpdateEmailTemplate is the resolver for the _update_email_template field.
func (r *mutationResolver) UpdateEmailTemplate(ctx context.Context, params model.UpdateEmailTemplateRequest) (*model.Response, error) {
	return resolvers.UpdateEmailTemplateResolver(ctx, params)
}

// DeleteEmailTemplate is the resolver for the _delete_email_template field.
func (r *mutationResolver) DeleteEmailTemplate(ctx context.Context, params model.DeleteEmailTemplateRequest) (*model.Response, error) {
	return resolvers.DeleteEmailTemplateResolver(ctx, params)
}

// Meta is the resolver for the meta field.
func (r *queryResolver) Meta(ctx context.Context) (*model.Meta, error) {
	return resolvers.MetaResolver(ctx)
}

// Session is the resolver for the session field.
func (r *queryResolver) Session(ctx context.Context, params *model.SessionQueryInput) (*model.AuthResponse, error) {
	return resolvers.SessionResolver(ctx, params)
}

// Profile is the resolver for the profile field.
func (r *queryResolver) Profile(ctx context.Context) (*model.User, error) {
	return resolvers.ProfileResolver(ctx)
}

// ValidateJwtToken is the resolver for the validate_jwt_token field.
func (r *queryResolver) ValidateJwtToken(ctx context.Context, params model.ValidateJWTTokenInput) (*model.ValidateJWTTokenResponse, error) {
	return resolvers.ValidateJwtTokenResolver(ctx, params)
}

// ValidateSession is the resolver for the validate_session field.
func (r *queryResolver) ValidateSession(ctx context.Context, params *model.ValidateSessionInput) (*model.ValidateSessionResponse, error) {
	return resolvers.ValidateSessionResolver(ctx, params)
}

// Users is the resolver for the _users field.
func (r *queryResolver) Users(ctx context.Context, params *model.PaginatedInput) (*model.Users, error) {
	return resolvers.UsersResolver(ctx, params)
}

// User is the resolver for the _user field.
func (r *queryResolver) User(ctx context.Context, params model.GetUserRequest) (*model.User, error) {
	return resolvers.UserResolver(ctx, params)
}

// VerificationRequests is the resolver for the _verification_requests field.
func (r *queryResolver) VerificationRequests(ctx context.Context, params *model.PaginatedInput) (*model.VerificationRequests, error) {
	return resolvers.VerificationRequestsResolver(ctx, params)
}

// AdminSession is the resolver for the _admin_session field.
func (r *queryResolver) AdminSession(ctx context.Context) (*model.Response, error) {
	return resolvers.AdminSessionResolver(ctx)
}

// Env is the resolver for the _env field.
func (r *queryResolver) Env(ctx context.Context) (*model.Env, error) {
	return resolvers.EnvResolver(ctx)
}

// Webhook is the resolver for the _webhook field.
func (r *queryResolver) Webhook(ctx context.Context, params model.WebhookRequest) (*model.Webhook, error) {
	return resolvers.WebhookResolver(ctx, params)
}

// Webhooks is the resolver for the _webhooks field.
func (r *queryResolver) Webhooks(ctx context.Context, params *model.PaginatedInput) (*model.Webhooks, error) {
	return resolvers.WebhooksResolver(ctx, params)
}

// WebhookLogs is the resolver for the _webhook_logs field.
func (r *queryResolver) WebhookLogs(ctx context.Context, params *model.ListWebhookLogRequest) (*model.WebhookLogs, error) {
	return resolvers.WebhookLogsResolver(ctx, params)
}

// EmailTemplates is the resolver for the _email_templates field.
func (r *queryResolver) EmailTemplates(ctx context.Context, params *model.PaginatedInput) (*model.EmailTemplates, error) {
	return resolvers.EmailTemplatesResolver(ctx, params)
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

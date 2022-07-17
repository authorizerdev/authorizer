package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/authorizerdev/authorizer/server/graph/generated"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/resolvers"
)

func (r *mutationResolver) Signup(ctx context.Context, params model.SignUpInput) (*model.AuthResponse, error) {
	return resolvers.SignupResolver(ctx, params)
}

func (r *mutationResolver) Login(ctx context.Context, params model.LoginInput) (*model.AuthResponse, error) {
	return resolvers.LoginResolver(ctx, params)
}

func (r *mutationResolver) MagicLinkLogin(ctx context.Context, params model.MagicLinkLoginInput) (*model.Response, error) {
	return resolvers.MagicLinkLoginResolver(ctx, params)
}

func (r *mutationResolver) Logout(ctx context.Context) (*model.Response, error) {
	return resolvers.LogoutResolver(ctx)
}

func (r *mutationResolver) UpdateProfile(ctx context.Context, params model.UpdateProfileInput) (*model.Response, error) {
	return resolvers.UpdateProfileResolver(ctx, params)
}

func (r *mutationResolver) VerifyEmail(ctx context.Context, params model.VerifyEmailInput) (*model.AuthResponse, error) {
	return resolvers.VerifyEmailResolver(ctx, params)
}

func (r *mutationResolver) ResendVerifyEmail(ctx context.Context, params model.ResendVerifyEmailInput) (*model.Response, error) {
	return resolvers.ResendVerifyEmailResolver(ctx, params)
}

func (r *mutationResolver) ForgotPassword(ctx context.Context, params model.ForgotPasswordInput) (*model.Response, error) {
	return resolvers.ForgotPasswordResolver(ctx, params)
}

func (r *mutationResolver) ResetPassword(ctx context.Context, params model.ResetPasswordInput) (*model.Response, error) {
	return resolvers.ResetPasswordResolver(ctx, params)
}

func (r *mutationResolver) Revoke(ctx context.Context, params model.OAuthRevokeInput) (*model.Response, error) {
	return resolvers.RevokeResolver(ctx, params)
}

func (r *mutationResolver) DeleteUser(ctx context.Context, params model.DeleteUserInput) (*model.Response, error) {
	return resolvers.DeleteUserResolver(ctx, params)
}

func (r *mutationResolver) UpdateUser(ctx context.Context, params model.UpdateUserInput) (*model.User, error) {
	return resolvers.UpdateUserResolver(ctx, params)
}

func (r *mutationResolver) AdminSignup(ctx context.Context, params model.AdminSignupInput) (*model.Response, error) {
	return resolvers.AdminSignupResolver(ctx, params)
}

func (r *mutationResolver) AdminLogin(ctx context.Context, params model.AdminLoginInput) (*model.Response, error) {
	return resolvers.AdminLoginResolver(ctx, params)
}

func (r *mutationResolver) AdminLogout(ctx context.Context) (*model.Response, error) {
	return resolvers.AdminLogoutResolver(ctx)
}

func (r *mutationResolver) UpdateEnv(ctx context.Context, params model.UpdateEnvInput) (*model.Response, error) {
	return resolvers.UpdateEnvResolver(ctx, params)
}

func (r *mutationResolver) InviteMembers(ctx context.Context, params model.InviteMemberInput) (*model.Response, error) {
	return resolvers.InviteMembersResolver(ctx, params)
}

func (r *mutationResolver) RevokeAccess(ctx context.Context, param model.UpdateAccessInput) (*model.Response, error) {
	return resolvers.RevokeAccessResolver(ctx, param)
}

func (r *mutationResolver) EnableAccess(ctx context.Context, param model.UpdateAccessInput) (*model.Response, error) {
	return resolvers.EnableAccessResolver(ctx, param)
}

func (r *mutationResolver) GenerateJwtKeys(ctx context.Context, params model.GenerateJWTKeysInput) (*model.GenerateJWTKeysResponse, error) {
	return resolvers.GenerateJWTKeysResolver(ctx, params)
}

func (r *mutationResolver) AddWebhook(ctx context.Context, params model.AddWebhookRequest) (*model.Response, error) {
	return resolvers.AddWebhookResolver(ctx, params)
}

func (r *mutationResolver) UpdateWebhook(ctx context.Context, params model.UpdateWebhookRequest) (*model.Response, error) {
	return resolvers.UpdateWebhookResolver(ctx, params)
}

func (r *mutationResolver) DeleteWebhook(ctx context.Context, params model.WebhookRequest) (*model.Response, error) {
	return resolvers.DeleteWebhookResolver(ctx, params)
}

func (r *mutationResolver) TestEndpoint(ctx context.Context, params model.TestEndpointRequest) (*model.TestEndpointResponse, error) {
	return resolvers.TestEndpointResolver(ctx, params)
}

func (r *mutationResolver) AddEmailTemplate(ctx context.Context, params model.AddEmailTemplateRequest) (*model.Response, error) {
	return resolvers.AddEmailTemplateResolver(ctx, params)
}

func (r *mutationResolver) UpdateEmailTemplate(ctx context.Context, params model.UpdateEmailTemplateRequest) (*model.Response, error) {
	return resolvers.UpdateEmailTemplateResolver(ctx, params)
}

func (r *mutationResolver) DeleteEmailTemplate(ctx context.Context, params model.DeleteEmailTemplateRequest) (*model.Response, error) {
	return resolvers.DeleteEmailTemplateResolver(ctx, params)
}

func (r *queryResolver) Meta(ctx context.Context) (*model.Meta, error) {
	return resolvers.MetaResolver(ctx)
}

func (r *queryResolver) Session(ctx context.Context, params *model.SessionQueryInput) (*model.AuthResponse, error) {
	return resolvers.SessionResolver(ctx, params)
}

func (r *queryResolver) Profile(ctx context.Context) (*model.User, error) {
	return resolvers.ProfileResolver(ctx)
}

func (r *queryResolver) ValidateJwtToken(ctx context.Context, params model.ValidateJWTTokenInput) (*model.ValidateJWTTokenResponse, error) {
	return resolvers.ValidateJwtTokenResolver(ctx, params)
}

func (r *queryResolver) Users(ctx context.Context, params *model.PaginatedInput) (*model.Users, error) {
	return resolvers.UsersResolver(ctx, params)
}

func (r *queryResolver) VerificationRequests(ctx context.Context, params *model.PaginatedInput) (*model.VerificationRequests, error) {
	return resolvers.VerificationRequestsResolver(ctx, params)
}

func (r *queryResolver) AdminSession(ctx context.Context) (*model.Response, error) {
	return resolvers.AdminSessionResolver(ctx)
}

func (r *queryResolver) Env(ctx context.Context) (*model.Env, error) {
	return resolvers.EnvResolver(ctx)
}

func (r *queryResolver) Webhook(ctx context.Context, params model.WebhookRequest) (*model.Webhook, error) {
	return resolvers.WebhookResolver(ctx, params)
}

func (r *queryResolver) Webhooks(ctx context.Context, params *model.PaginatedInput) (*model.Webhooks, error) {
	return resolvers.WebhooksResolver(ctx, params)
}

func (r *queryResolver) WebhookLogs(ctx context.Context, params *model.ListWebhookLogRequest) (*model.WebhookLogs, error) {
	return resolvers.WebhookLogsResolver(ctx, params)
}

func (r *queryResolver) EmailTemplates(ctx context.Context, params *model.PaginatedInput) (*model.EmailTemplates, error) {
	return resolvers.EmailTemplatesResolver(ctx, params)
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type (
	mutationResolver struct{ *Resolver }
	queryResolver    struct{ *Resolver }
)

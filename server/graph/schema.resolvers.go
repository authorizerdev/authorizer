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
	return resolvers.Signup(ctx, params)
}

func (r *mutationResolver) Login(ctx context.Context, params model.LoginInput) (*model.AuthResponse, error) {
	return resolvers.Login(ctx, params)
}

func (r *mutationResolver) MagicLinkLogin(ctx context.Context, params model.MagicLinkLoginInput) (*model.Response, error) {
	return resolvers.MagicLinkLogin(ctx, params)
}

func (r *mutationResolver) Logout(ctx context.Context) (*model.Response, error) {
	return resolvers.Logout(ctx)
}

func (r *mutationResolver) UpdateProfile(ctx context.Context, params model.UpdateProfileInput) (*model.Response, error) {
	return resolvers.UpdateProfile(ctx, params)
}

func (r *mutationResolver) VerifyEmail(ctx context.Context, params model.VerifyEmailInput) (*model.AuthResponse, error) {
	return resolvers.VerifyEmail(ctx, params)
}

func (r *mutationResolver) ResendVerifyEmail(ctx context.Context, params model.ResendVerifyEmailInput) (*model.Response, error) {
	return resolvers.ResendVerifyEmail(ctx, params)
}

func (r *mutationResolver) ForgotPassword(ctx context.Context, params model.ForgotPasswordInput) (*model.Response, error) {
	return resolvers.ForgotPassword(ctx, params)
}

func (r *mutationResolver) ResetPassword(ctx context.Context, params model.ResetPasswordInput) (*model.Response, error) {
	return resolvers.ResetPassword(ctx, params)
}

func (r *mutationResolver) DeleteUser(ctx context.Context, params model.DeleteUserInput) (*model.Response, error) {
	return resolvers.DeleteUser(ctx, params)
}

func (r *mutationResolver) UpdateUser(ctx context.Context, params model.UpdateUserInput) (*model.User, error) {
	return resolvers.UpdateUser(ctx, params)
}

func (r *mutationResolver) AdminSignup(ctx context.Context, params model.AdminLoginInput) (*model.AdminLoginResponse, error) {
	return resolvers.AdminSignupResolver(ctx, params)
}

func (r *mutationResolver) AdminLogin(ctx context.Context, params model.AdminLoginInput) (*model.AdminLoginResponse, error) {
	return resolvers.AdminLoginResolver(ctx, params)
}

func (r *mutationResolver) AdminLogout(ctx context.Context) (*model.Response, error) {
	return resolvers.AdminLogout(ctx)
}

func (r *mutationResolver) UpdateConfig(ctx context.Context, params model.UpdateConfigInput) (*model.Response, error) {
	return resolvers.UpdateConfigResolver(ctx, params)
}

func (r *queryResolver) Meta(ctx context.Context) (*model.Meta, error) {
	return resolvers.Meta(ctx)
}

func (r *queryResolver) Session(ctx context.Context, roles []string) (*model.AuthResponse, error) {
	return resolvers.Session(ctx, roles)
}

func (r *queryResolver) Profile(ctx context.Context) (*model.User, error) {
	return resolvers.Profile(ctx)
}

func (r *queryResolver) Users(ctx context.Context) ([]*model.User, error) {
	return resolvers.Users(ctx)
}

func (r *queryResolver) VerificationRequests(ctx context.Context) ([]*model.VerificationRequest, error) {
	return resolvers.VerificationRequests(ctx)
}

func (r *queryResolver) AdminSession(ctx context.Context) (*model.AdminLoginResponse, error) {
	return resolvers.AdminSession(ctx)
}

func (r *queryResolver) Config(ctx context.Context) (*model.Config, error) {
	return resolvers.ConfigResolver(ctx)
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

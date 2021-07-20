package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	"github.com/yauthdev/yauth/server/graph/generated"
	"github.com/yauthdev/yauth/server/graph/model"
	"github.com/yauthdev/yauth/server/resolvers"
)

func (r *mutationResolver) Signup(ctx context.Context, params model.SignUpInput) (*model.Response, error) {
	return resolvers.Signup(ctx, params)
}

func (r *mutationResolver) Login(ctx context.Context, params model.LoginInput) (*model.LoginResponse, error) {
	return resolvers.Login(ctx, params)
}

func (r *mutationResolver) Logout(ctx context.Context) (*model.Response, error) {
	return resolvers.Logout(ctx)
}

func (r *mutationResolver) UpdateProfile(ctx context.Context, params model.UpdateProfileInput) (*model.Response, error) {
	return resolvers.UpdateProfile(ctx, params)
}

func (r *mutationResolver) VerifyEmail(ctx context.Context, params model.VerifyEmailInput) (*model.LoginResponse, error) {
	return resolvers.VerifyEmail(ctx, params)
}

func (r *mutationResolver) ResendVerifyEmail(ctx context.Context, params model.ResendVerifyEmailInput) (*model.Response, error) {
	return resolvers.ResendVerifyEmail(ctx, params)
}

func (r *mutationResolver) ForgotPasswordRequest(ctx context.Context, params model.ForgotPasswordRequestInput) (*model.Response, error) {
	return resolvers.ForgotPasswordRequest(ctx, params)
}

func (r *mutationResolver) ForgotPassword(ctx context.Context, params model.ForgotPasswordInput) (*model.Response, error) {
	return resolvers.ForgotPassword(ctx, params)
}

func (r *queryResolver) Users(ctx context.Context) ([]*model.User, error) {
	return resolvers.Users(ctx)
}

func (r *queryResolver) Token(ctx context.Context) (*model.LoginResponse, error) {
	return resolvers.Token(ctx)
}

func (r *queryResolver) Profile(ctx context.Context) (*model.User, error) {
	return resolvers.Profile(ctx)
}

func (r *queryResolver) VerificationRequests(ctx context.Context) ([]*model.VerificationRequest, error) {
	return resolvers.VerificationRequests(ctx)
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

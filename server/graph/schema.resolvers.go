package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/yauthdev/yauth/server/db"
	"github.com/yauthdev/yauth/server/graph/generated"
	"github.com/yauthdev/yauth/server/graph/model"
	"github.com/yauthdev/yauth/server/utils"
)

func (r *mutationResolver) BasicAuthSignUp(ctx context.Context, params model.BasicAuthSignupInput) (*model.BasicAuthSignupResponse, error) {
	var res *model.BasicAuthSignupResponse
	if params.CofirmPassword != params.Password {
		res = &model.BasicAuthSignupResponse{
			Success:    false,
			Message:    `Passowrd and Confirm Password does not match`,
			StatusCode: 400,
			Errors: []*model.Error{&model.Error{
				Message: `Passowrd and Confirm Password does not match`,
				Reason:  `password and confirm_password fields should match`,
			}},
		}
	}

	params.Email = strings.ToLower(params.Email)

	if !utils.IsValidEmail(params.Email) {
		res = &model.BasicAuthSignupResponse{
			Success:    false,
			Message:    `Invalid email address`,
			StatusCode: 400,
			Errors: []*model.Error{&model.Error{
				Message: `Invalid email address`,
				Reason:  `invalid email address`,
			}},
		}
	}

	// find user with email
	existingUser, err := db.Mgr.GetUserByEmail(params.Email)
	if err != nil {
		log.Println("User with email " + params.Email + " not found")
	}

	if existingUser.EmailVerifiedAt > 0 {
		// email is verified
		res = &model.BasicAuthSignupResponse{
			Success:    false,
			Message:    `You have already signed up. Please login`,
			StatusCode: 400,
			Errors: []*model.Error{&model.Error{
				Message: `Already signed up`,
				Reason:  `already signed up`,
			}},
		}
	} else {
		user := db.User{
			Email:    params.Email,
			Password: params.Password,
		}

		if params.FirstName != nil {
			user.FirstName = *params.FirstName
		}

		if params.LastName != nil {
			user.LastName = *params.LastName
		}

		_, err = db.Mgr.AddUser(user)
		if err != nil {
			return res, err
		}

		// insert verification request
		verificationType := "BASIC_AUTH_SIGNUP"
		token, err := utils.CreateVerificationToken(params.Email, verificationType)
		if err != nil {
			log.Println(`Error generating token`, err)
		}
		db.Mgr.AddVerification(db.Verification{
			Token:      token,
			Identifier: verificationType,
			ExpiresAt:  time.Now().Add(time.Minute * 30).Unix(),
			Email:      params.Email,
		})

		// exec it as go routin so that we can reduce the api latency
		go func() {
			utils.SendVerificationMail(params.Email, token)
		}()

		res = &model.BasicAuthSignupResponse{
			Success:    true,
			Message:    `Verification email sent successfully. Please check your inbox`,
			StatusCode: 200,
		}
	}

	return res, nil
}

func (r *mutationResolver) BasicAuthLogin(ctx context.Context, params model.BasicAuthLoginInput) (*model.BasicAuthLoginResponse, error) {
	panic(fmt.Errorf("not implemented"))
}

func (r *queryResolver) Users(ctx context.Context) ([]*model.User, error) {
	var res []*model.User
	users, err := db.Mgr.GetUsers()
	if err != nil {
		return res, err
	}

	for _, user := range users {
		res = append(res, &model.User{
			ID:              fmt.Sprintf("%d", user.ID),
			Email:           user.Email,
			SignUpMethod:    user.SignupMethod,
			FirstName:       &user.FirstName,
			LastName:        &user.LastName,
			Password:        &user.Password,
			EmailVerifiedAt: &user.EmailVerifiedAt,
		})
	}

	return res, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type (
	mutationResolver struct{ *Resolver }
	queryResolver    struct{ *Resolver }
)

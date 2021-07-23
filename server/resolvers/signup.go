package resolvers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/utils"
)

func Signup(ctx context.Context, params model.SignUpInput) (*model.Response, error) {
	var res *model.Response
	if params.ConfirmPassword != params.Password {
		return res, fmt.Errorf(`passowrd and confirm password does not match`)
	}

	params.Email = strings.ToLower(params.Email)

	if !utils.IsValidEmail(params.Email) {
		return res, fmt.Errorf(`invalid email address`)
	}

	// find user with email
	existingUser, err := db.Mgr.GetUserByEmail(params.Email)
	if err != nil {
		log.Println("User with email " + params.Email + " not found")
	}

	if existingUser.EmailVerifiedAt > 0 {
		// email is verified
		return res, fmt.Errorf(`you have already signed up. Please login`)
	}
	user := db.User{
		Email: params.Email,
	}

	password, _ := utils.HashPassword(params.Password)
	user.Password = password

	if params.FirstName != nil {
		user.FirstName = *params.FirstName
	}

	if params.LastName != nil {
		user.LastName = *params.LastName
	}

	user.SignupMethod = enum.BasicAuth.String()
	_, err = db.Mgr.SaveUser(user)
	if err != nil {
		return res, err
	}

	// insert verification request
	verificationType := enum.BasicAuthSignup.String()
	token, err := utils.CreateVerificationToken(params.Email, verificationType)
	if err != nil {
		log.Println(`Error generating token`, err)
	}
	db.Mgr.AddVerification(db.VerificationRequest{
		Token:      token,
		Identifier: verificationType,
		ExpiresAt:  time.Now().Add(time.Minute * 30).Unix(),
		Email:      params.Email,
	})

	// exec it as go routin so that we can reduce the api latency
	go func() {
		utils.SendVerificationMail(params.Email, token)
	}()

	res = &model.Response{
		Message: `Verification email has been sent. Please check your inbox`,
	}

	return res, nil
}

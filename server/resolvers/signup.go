package resolvers

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/yauthdev/yauth/server/db"
	"github.com/yauthdev/yauth/server/enum"
	"github.com/yauthdev/yauth/server/graph/model"
	"github.com/yauthdev/yauth/server/utils"
)

func Signup(ctx context.Context, params model.SignUpInput) (*model.SignUpResponse, error) {
	var res *model.SignUpResponse
	if params.CofirmPassword != params.Password {
		return res, errors.New(`Passowrd and Confirm Password does not match`)
	}

	params.Email = strings.ToLower(params.Email)

	if !utils.IsValidEmail(params.Email) {
		return res, errors.New(`Invalid email address`)
	}

	// find user with email
	existingUser, err := db.Mgr.GetUserByEmail(params.Email)
	if err != nil {
		log.Println("User with email " + params.Email + " not found")
	}

	if existingUser.EmailVerifiedAt > 0 {
		// email is verified
		return res, errors.New(`You have already signed up. Please login`)
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
	verificationType := enum.BasicAuth.String()
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

	res = &model.SignUpResponse{
		Message: `Verification email sent successfully. Please check your inbox`,
	}

	return res, nil
}

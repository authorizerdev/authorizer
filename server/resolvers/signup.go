package resolvers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
)

func Signup(ctx context.Context, params model.SignUpInput) (*model.AuthResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.AuthResponse
	if err != nil {
		return res, err
	}

	if constants.DISABLE_BASIC_AUTHENTICATION == "true" {
		return res, fmt.Errorf(`basic authentication is disabled for this instance`)
	}
	if params.ConfirmPassword != params.Password {
		return res, fmt.Errorf(`passowrd and confirm password does not match`)
	}

	params.Email = strings.ToLower(params.Email)

	if !utils.IsValidEmail(params.Email) {
		return res, fmt.Errorf(`invalid email address`)
	}

	inputRoles := []string{}

	if len(params.Roles) > 0 {
		// check if roles exists
		if !utils.IsValidRoles(constants.ROLES, params.Roles) {
			return res, fmt.Errorf(`invalid roles`)
		} else {
			inputRoles = params.Roles
		}
	} else {
		inputRoles = constants.DEFAULT_ROLES
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

	user.Roles = strings.Join(inputRoles, ",")

	password, _ := utils.HashPassword(params.Password)
	user.Password = password

	if params.FirstName != nil {
		user.FirstName = *params.FirstName
	}

	if params.LastName != nil {
		user.LastName = *params.LastName
	}

	user.SignupMethod = enum.BasicAuth.String()
	if constants.DISABLE_EMAIL_VERIFICATION == "true" {
		user.EmailVerifiedAt = time.Now().Unix()
	}
	user, err = db.Mgr.SaveUser(user)
	if err != nil {
		return res, err
	}
	userIdStr := fmt.Sprintf("%v", user.ID)
	roles := strings.Split(user.Roles, ",")
	userToReturn := &model.User{
		ID:              userIdStr,
		Email:           user.Email,
		Image:           &user.Image,
		FirstName:       &user.FirstName,
		LastName:        &user.LastName,
		SignupMethod:    user.SignupMethod,
		EmailVerifiedAt: &user.EmailVerifiedAt,
		Roles:           strings.Split(user.Roles, ","),
		CreatedAt:       &user.CreatedAt,
		UpdatedAt:       &user.UpdatedAt,
	}

	if constants.DISABLE_EMAIL_VERIFICATION != "true" {
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

		res = &model.AuthResponse{
			Message: `Verification email has been sent. Please check your inbox`,
			User:    userToReturn,
		}
	} else {

		refreshToken, _, _ := utils.CreateAuthToken(user, enum.RefreshToken, roles)

		accessToken, expiresAt, _ := utils.CreateAuthToken(user, enum.AccessToken, roles)

		session.SetToken(userIdStr, refreshToken)
		res = &model.AuthResponse{
			Message:              `Signed up successfully.`,
			AccessToken:          &accessToken,
			AccessTokenExpiresAt: &expiresAt,
			User:                 userToReturn,
		}

		utils.SetCookie(gc, accessToken)
	}

	return res, nil
}

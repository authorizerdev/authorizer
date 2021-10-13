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

func AdminUpdateUser(ctx context.Context, params model.AdminUpdateUserInput) (*model.User, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.User
	if err != nil {
		return res, err
	}

	if !utils.IsSuperAdmin(gc) {
		return res, fmt.Errorf("unauthorized")
	}

	if params.FirstName == nil && params.LastName == nil && params.Image == nil && params.Email == nil && params.Roles == nil {
		return res, fmt.Errorf("please enter atleast one param to update")
	}

	user, err := db.Mgr.GetUserByID(params.ID)
	if err != nil {
		return res, fmt.Errorf(`User not found`)
	}

	if params.FirstName != nil && user.FirstName != *params.FirstName {
		user.FirstName = *params.FirstName
	}

	if params.LastName != nil && user.LastName != *params.LastName {
		user.LastName = *params.LastName
	}

	if params.Image != nil && user.Image != *params.Image {
		user.Image = *params.Image
	}

	if params.Email != nil && user.Email != *params.Email {
		// check if valid email
		if !utils.IsValidEmail(*params.Email) {
			return res, fmt.Errorf("invalid email address")
		}
		newEmail := strings.ToLower(*params.Email)
		// check if user with new email exists
		_, err = db.Mgr.GetUserByEmail(newEmail)
		// err = nil means user exists
		if err == nil {
			return res, fmt.Errorf("user with this email address already exists")
		}

		session.DeleteToken(fmt.Sprintf("%v", user.ID))
		utils.DeleteCookie(gc)

		user.Email = newEmail
		user.EmailVerifiedAt = 0
		// insert verification request
		verificationType := enum.UpdateEmail.String()
		token, err := utils.CreateVerificationToken(newEmail, verificationType)
		if err != nil {
			log.Println(`Error generating token`, err)
		}
		db.Mgr.AddVerification(db.VerificationRequest{
			Token:      token,
			Identifier: verificationType,
			ExpiresAt:  time.Now().Add(time.Minute * 30).Unix(),
			Email:      newEmail,
		})

		// exec it as go routin so that we can reduce the api latency
		go func() {
			utils.SendVerificationMail(newEmail, token)
		}()
	}

	rolesToSave := ""
	if params.Roles != nil && len(params.Roles) > 0 {
		currentRoles := strings.Split(user.Roles, ",")
		inputRoles := []string{}
		for _, item := range params.Roles {
			inputRoles = append(inputRoles, *item)
		}

		if !utils.IsValidRoles(append([]string{}, append(constants.ROLES, constants.PROTECTED_ROLES...)...), inputRoles) {
			return res, fmt.Errorf("invalid list of roles")
		}

		if !utils.IsStringArrayEqual(inputRoles, currentRoles) {
			rolesToSave = strings.Join(inputRoles, ",")
		}

		session.DeleteToken(fmt.Sprintf("%v", user.ID))
		utils.DeleteCookie(gc)
	}

	if rolesToSave != "" {
		user.Roles = rolesToSave
	}

	user, err = db.Mgr.UpdateUser(user)
	if err != nil {
		log.Println("Error updating user:", err)
		return res, err
	}

	res = &model.User{
		ID:        params.ID,
		Email:     user.Email,
		Image:     &user.Image,
		FirstName: &user.FirstName,
		LastName:  &user.LastName,
		Roles:     strings.Split(user.Roles, ","),
		CreatedAt: &user.CreatedAt,
		UpdatedAt: &user.UpdatedAt,
	}
	return res, nil
}

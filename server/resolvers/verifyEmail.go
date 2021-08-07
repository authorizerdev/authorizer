package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/enum"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
)

func VerifyEmail(ctx context.Context, params model.VerifyEmailInput) (*model.AuthResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.AuthResponse
	if err != nil {
		return res, err
	}

	_, err = db.Mgr.GetVerificationByToken(params.Token)
	if err != nil {
		return res, fmt.Errorf(`invalid token`)
	}

	// verify if token exists in db
	claim, err := utils.VerifyVerificationToken(params.Token)
	if err != nil {
		return res, fmt.Errorf(`invalid token`)
	}

	user, err := db.Mgr.GetUserByEmail(claim.Email)
	if err != nil {
		return res, err
	}

	// update email_verified_at in users table
	db.Mgr.UpdateVerificationTime(time.Now().Unix(), user.ID)
	// delete from verification table
	db.Mgr.DeleteToken(claim.Email)

	userIdStr := fmt.Sprintf("%v", user.ID)
	refreshToken, _, _ := utils.CreateAuthToken(utils.UserAuthInfo{
		ID:    userIdStr,
		Email: user.Email,
	}, enum.RefreshToken)

	accessToken, expiresAt, _ := utils.CreateAuthToken(utils.UserAuthInfo{
		ID:    userIdStr,
		Email: user.Email,
	}, enum.AccessToken)

	session.SetToken(userIdStr, refreshToken)

	res = &model.AuthResponse{
		Message:              `Email verified successfully.`,
		AccessToken:          &accessToken,
		AccessTokenExpiresAt: &expiresAt,
		User: &model.User{
			ID:              userIdStr,
			Email:           user.Email,
			Image:           &user.Image,
			FirstName:       &user.FirstName,
			LastName:        &user.LastName,
			SignupMethod:    user.SignupMethod,
			EmailVerifiedAt: &user.EmailVerifiedAt,
			CreatedAt:       &user.CreatedAt,
			UpdatedAt:       &user.UpdatedAt,
		},
	}

	utils.SetCookie(gc, accessToken)

	return res, nil
}

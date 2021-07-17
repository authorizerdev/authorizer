package resolvers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yauthdev/yauth/server/db"
	"github.com/yauthdev/yauth/server/enum"
	"github.com/yauthdev/yauth/server/graph/model"
	"github.com/yauthdev/yauth/server/session"
	"github.com/yauthdev/yauth/server/utils"
)

func VerifySignupToken(ctx context.Context, params model.VerifySignupTokenInput) (*model.LoginResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.LoginResponse
	if err != nil {
		return res, err
	}

	_, err = db.Mgr.GetVerificationByToken(params.Token)
	if err != nil {
		return res, errors.New(`Invalid token`)
	}

	// verify if token exists in db
	claim, err := utils.VerifyVerificationToken(params.Token)
	if err != nil {
		return res, errors.New(`Invalid token`)
	}

	user, err := db.Mgr.GetUserByEmail(claim.Email)
	if err != nil {
		return res, err
	}

	// update email_verified_at in users table
	db.Mgr.UpdateVerificationTime(time.Now().Unix(), user.ID)
	// delete from verification table
	db.Mgr.DeleteToken(claim.Email)

	userIdStr := fmt.Sprintf("%d", user.ID)
	refreshToken, _, _ := utils.CreateAuthToken(utils.UserAuthInfo{
		ID:    userIdStr,
		Email: user.Email,
	}, enum.RefreshToken)

	accessToken, expiresAt, _ := utils.CreateAuthToken(utils.UserAuthInfo{
		ID:    userIdStr,
		Email: user.Email,
	}, enum.AccessToken)

	session.SetToken(userIdStr, refreshToken)

	res = &model.LoginResponse{
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
		},
	}

	utils.SetCookie(gc, accessToken)

	return res, nil
}

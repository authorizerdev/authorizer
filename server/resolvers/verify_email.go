package resolvers

import (
	"context"
	"fmt"
	"strings"
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

	verificationRequest, err := db.Mgr.GetVerificationByToken(params.Token)
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
	now := time.Now().Unix()
	user.EmailVerifiedAt = &now
	db.Mgr.UpdateUser(user)
	// delete from verification table
	db.Mgr.DeleteVerificationRequest(verificationRequest)

	roles := strings.Split(user.Roles, ",")
	refreshToken, _, _ := utils.CreateAuthToken(user, enum.RefreshToken, roles)

	accessToken, expiresAt, _ := utils.CreateAuthToken(user, enum.AccessToken, roles)

	session.SetToken(user.ID, accessToken, refreshToken)
	utils.CreateSession(user.ID, gc)

	res = &model.AuthResponse{
		Message:     `Email verified successfully.`,
		AccessToken: &accessToken,
		ExpiresAt:   &expiresAt,
		User:        utils.GetResUser(user),
	}

	utils.SetCookie(gc, accessToken)

	return res, nil
}

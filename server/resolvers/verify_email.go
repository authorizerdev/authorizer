package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/constants"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/session"
	"github.com/authorizerdev/authorizer/server/utils"
)

// VerifyEmailResolver is a resolver for verify email mutation
func VerifyEmailResolver(ctx context.Context, params model.VerifyEmailInput) (*model.AuthResponse, error) {
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
	refreshToken, _, _ := utils.CreateAuthToken(user, constants.TokenTypeRefreshToken, roles)
	accessToken, expiresAt, _ := utils.CreateAuthToken(user, constants.TokenTypeAccessToken, roles)

	session.SetUserSession(user.ID, accessToken, refreshToken)
	utils.SaveSessionInDB(user.ID, gc)

	res = &model.AuthResponse{
		Message:     `Email verified successfully.`,
		AccessToken: &accessToken,
		ExpiresAt:   &expiresAt,
		User:        utils.GetResponseUserData(user),
	}

	utils.SetCookie(gc, accessToken)

	return res, nil
}

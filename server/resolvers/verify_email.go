package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// VerifyEmailResolver is a resolver for verify email mutation
func VerifyEmailResolver(ctx context.Context, params model.VerifyEmailInput) (*model.AuthResponse, error) {
	gc, err := utils.GinContextFromContext(ctx)
	var res *model.AuthResponse
	if err != nil {
		return res, err
	}

	verificationRequest, err := db.Provider.GetVerificationRequestByToken(params.Token)
	if err != nil {
		return res, fmt.Errorf(`invalid token`)
	}

	// verify if token exists in db
	claim, err := token.ParseJWTToken(params.Token)
	if err != nil {
		return res, fmt.Errorf(`invalid token`)
	}

	user, err := db.Provider.GetUserByEmail(claim["email"].(string))
	if err != nil {
		return res, err
	}

	// update email_verified_at in users table
	now := time.Now().Unix()
	user.EmailVerifiedAt = &now
	db.Provider.UpdateUser(user)
	// delete from verification table
	db.Provider.DeleteVerificationRequest(verificationRequest)

	roles := strings.Split(user.Roles, ",")
	authToken, err := token.CreateAuthToken(user, roles)
	if err != nil {
		return res, err
	}
	sessionstore.SetUserSession(user.ID, authToken.FingerPrint, authToken.RefreshToken.Token)
	cookie.SetCookie(gc, authToken.AccessToken.Token, authToken.RefreshToken.Token, authToken.FingerPrintHash)
	utils.SaveSessionInDB(user.ID, gc)

	res = &model.AuthResponse{
		Message:     `Email verified successfully.`,
		AccessToken: &authToken.AccessToken.Token,
		ExpiresAt:   &authToken.AccessToken.ExpiresAt,
		User:        user.AsAPIUser(),
	}

	return res, nil
}

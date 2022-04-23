package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
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
		return res, fmt.Errorf(`invalid token: %s`, err.Error())
	}

	// verify if token exists in db
	hostname := utils.GetHost(gc)
	claim, err := token.ParseJWTToken(params.Token, hostname, verificationRequest.Nonce, verificationRequest.Email)
	if err != nil {
		return res, fmt.Errorf(`invalid token: %s`, err.Error())
	}

	user, err := db.Provider.GetUserByEmail(claim["sub"].(string))
	if err != nil {
		return res, err
	}

	// update email_verified_at in users table
	now := time.Now().Unix()
	user.EmailVerifiedAt = &now
	user, err = db.Provider.UpdateUser(user)
	if err != nil {
		return res, err
	}
	// delete from verification table
	err = db.Provider.DeleteVerificationRequest(verificationRequest)
	if err != nil {
		return res, err
	}

	roles := strings.Split(user.Roles, ",")
	scope := []string{"openid", "email", "profile"}
	authToken, err := token.CreateAuthToken(gc, user, roles, scope)
	if err != nil {
		return res, err
	}

	sessionstore.SetState(authToken.FingerPrintHash, authToken.FingerPrint+"@"+user.ID)
	sessionstore.SetState(authToken.AccessToken.Token, authToken.FingerPrint+"@"+user.ID)
	cookie.SetSession(gc, authToken.FingerPrintHash)
	go db.Provider.AddSession(models.Session{
		UserID:    user.ID,
		UserAgent: utils.GetUserAgent(gc.Request),
		IP:        utils.GetIP(gc.Request),
	})

	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res = &model.AuthResponse{
		Message:     `Email verified successfully.`,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &expiresIn,
		User:        user.AsAPIUser(),
	}
	return res, nil
}

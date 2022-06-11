package resolvers

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/db/models"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/memorystore"
	"github.com/authorizerdev/authorizer/server/parsers"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// VerifyEmailResolver is a resolver for verify email mutation
func VerifyEmailResolver(ctx context.Context, params model.VerifyEmailInput) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug("Failed to get GinContext: ", err)
		return res, err
	}

	verificationRequest, err := db.Provider.GetVerificationRequestByToken(params.Token)
	if err != nil {
		log.Debug("Failed to get verification request by token: ", err)
		return res, fmt.Errorf(`invalid token: %s`, err.Error())
	}

	// verify if token exists in db
	hostname := parsers.GetHost(gc)
	claim, err := token.ParseJWTToken(params.Token)
	if err != nil {
		log.Debug("Failed to parse token: ", err)
		return res, fmt.Errorf(`invalid token: %s`, err.Error())
	}

	if ok, err := token.ValidateJWTClaims(claim, hostname, verificationRequest.Nonce, verificationRequest.Email); !ok || err != nil {
		log.Debug("Failed to validate jwt claims: ", err)
		return res, fmt.Errorf(`invalid token: %s`, err.Error())
	}

	email := claim["sub"].(string)
	log := log.WithFields(log.Fields{
		"email": email,
	})
	user, err := db.Provider.GetUserByEmail(email)
	if err != nil {
		log.Debug("Failed to get user by email: ", err)
		return res, err
	}

	// update email_verified_at in users table
	now := time.Now().Unix()
	user.EmailVerifiedAt = &now
	user, err = db.Provider.UpdateUser(user)
	if err != nil {
		log.Debug("Failed to update user: ", err)
		return res, err
	}
	// delete from verification table
	err = db.Provider.DeleteVerificationRequest(verificationRequest)
	if err != nil {
		log.Debug("Failed to delete verification request: ", err)
		return res, err
	}

	roles := strings.Split(user.Roles, ",")
	scope := []string{"openid", "email", "profile"}
	authToken, err := token.CreateAuthToken(gc, user, roles, scope)
	if err != nil {
		log.Debug("Failed to create auth token: ", err)
		return res, err
	}

	memorystore.Provider.SetUserSession(user.ID, authToken.FingerPrintHash, authToken.FingerPrint)
	memorystore.Provider.SetUserSession(user.ID, authToken.AccessToken.Token, authToken.FingerPrint)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		memorystore.Provider.SetUserSession(user.ID, authToken.RefreshToken.Token, authToken.FingerPrint)
	}

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

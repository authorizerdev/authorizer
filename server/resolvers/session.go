package resolvers

import (
	"context"
	"fmt"
	"time"

	"github.com/authorizerdev/authorizer/server/cookie"
	"github.com/authorizerdev/authorizer/server/db"
	"github.com/authorizerdev/authorizer/server/graph/model"
	"github.com/authorizerdev/authorizer/server/sessionstore"
	"github.com/authorizerdev/authorizer/server/token"
	"github.com/authorizerdev/authorizer/server/utils"
)

// SessionResolver is a resolver for session query
// TODO allow validating with code and code verifier instead of cookie (PKCE flow)
func SessionResolver(ctx context.Context, params *model.SessionQueryInput) (*model.AuthResponse, error) {
	var res *model.AuthResponse

	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		return res, err
	}

	sessionToken, err := cookie.GetSession(gc)
	if err != nil {
		return res, err
	}

	// get session from cookie
	claims, err := token.ValidateBrowserSession(gc, sessionToken)
	if err != nil {
		return res, err
	}
	userID := claims.Subject
	user, err := db.Provider.GetUserByID(userID)
	if err != nil {
		return res, err
	}

	// refresh token has "roles" as claim
	claimRoleInterface := claims.Roles
	claimRoles := []string{}
	for _, v := range claimRoleInterface {
		claimRoles = append(claimRoles, v)
	}

	if params != nil && params.Roles != nil && len(params.Roles) > 0 {
		for _, v := range params.Roles {
			if !utils.StringSliceContains(claimRoles, v) {
				return res, fmt.Errorf(`unauthorized`)
			}
		}
	}

	scope := []string{"openid", "email", "profile"}
	if params != nil && params.Scope != nil && len(scope) > 0 {
		scope = params.Scope
	}

	authToken, err := token.CreateAuthToken(gc, user, claimRoles, scope)
	if err != nil {
		return res, err
	}

	// rollover the session for security
	sessionstore.RemoveState(sessionToken)
	sessionstore.SetState(authToken.FingerPrintHash, authToken.FingerPrint+"@"+user.ID)
	sessionstore.SetState(authToken.AccessToken.Token, authToken.FingerPrint+"@"+user.ID)
	cookie.SetSession(gc, authToken.FingerPrintHash)

	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res = &model.AuthResponse{
		Message:     `Session token refreshed`,
		AccessToken: &authToken.AccessToken.Token,
		ExpiresIn:   &expiresIn,
		IDToken:     &authToken.IDToken.Token,
		User:        user.AsAPIUser(),
	}

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		sessionstore.SetState(authToken.RefreshToken.Token, authToken.FingerPrint+"@"+user.ID)
	}

	return res, nil
}

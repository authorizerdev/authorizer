package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Session is the method to get session.
// It also refreshes the session token.
// TODO allow validating with code and code verifier instead of cookie (PKCE flow)
func (s *service) Session(ctx context.Context, params *model.SessionQueryInput) (*model.AuthResponse, error) {
	log := s.Log.With().Str("func", "Session").Logger()
	gc, err := utils.GinContextFromContext(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get GinContext")
		return nil, err
	}

	sessionToken, err := cookie.GetSession(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get session token")
		return nil, errors.New("unauthorized")
	}

	// get session from cookie
	claims, err := s.TokenProvider.ValidateBrowserSession(gc, sessionToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to validate session token")
		return nil, errors.New("unauthorized")
	}
	userID := claims.Subject
	log = log.With().Str("user_id", userID).Logger()
	user, err := s.StorageProvider.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user")
		return nil, err
	}

	// refresh token has "roles" as claim
	claimRoleInterface := claims.Roles
	claimRoles := []string{}
	claimRoles = append(claimRoles, claimRoleInterface...)

	if params != nil && params.Roles != nil && len(params.Roles) > 0 {
		for _, v := range params.Roles {
			if !utils.StringSliceContains(claimRoles, v) {
				log.Debug().Msg("User does not have required role")
				return nil, fmt.Errorf(`unauthorized`)
			}
		}
	}

	scope := []string{"openid", "email", "profile"}
	if params != nil && params.Scope != nil && len(scope) > 0 {
		scope = params.Scope
	}

	nonce := uuid.New().String()
	hostname := parsers.GetHost(gc)
	//  user, claimRoles, scope, claims.LoginMethod, nonce, ""
	authToken, err := s.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
		User:        user,
		Nonce:       nonce,
		Roles:       claimRoles,
		Scope:       scope,
		LoginMethod: claims.LoginMethod,
		HostName:    hostname,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to CreateAuthToken")
		return nil, err
	}

	// rollover the session for security
	sessionKey := userID
	if claims.LoginMethod != "" {
		sessionKey = claims.LoginMethod + ":" + userID
	}
	go s.MemoryStoreProvider.DeleteUserSession(sessionKey, claims.Nonce)

	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res := &model.AuthResponse{
		Message:     `Session token refreshed`,
		AccessToken: &authToken.AccessToken.Token,
		ExpiresIn:   &expiresIn,
		IDToken:     &authToken.IDToken.Token,
		User:        user.AsAPIUser(),
	}

	cookie.SetSession(gc, authToken.FingerPrintHash)
	s.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	s.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		s.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}
	return res, nil
}

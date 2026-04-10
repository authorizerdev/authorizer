package graphql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/parsers"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Session is the method to get session.
// It also refreshes the session token.
// TODO allow validating with code and code verifier instead of cookie (PKCE flow)
func (g *graphqlProvider) Session(ctx context.Context, params *model.SessionQueryRequest) (*model.AuthResponse, error) {
	log := g.Log.With().Str("func", "Session").Logger()
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
	claims, err := g.TokenProvider.ValidateBrowserSession(gc, sessionToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to validate session token")
		return nil, errors.New("unauthorized")
	}
	userID := claims.Subject
	log = log.With().Str("user_id", userID).Logger()
	user, err := g.StorageProvider.GetUserByID(ctx, userID)
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
	if params != nil && params.Scope != nil && len(params.Scope) > 0 {
		scope = params.Scope
	}

	// OIDC authorize flow: if state is provided, consume the authorize state
	// and prepare code/challenge data so the authorization code can be stored
	// after token creation. This handles the case where the login UI auto-detects
	// an existing session (e.g., prompt=login forced re-auth at /authorize but
	// the session cookie is still valid for GraphQL queries).
	code := ""
	codeChallenge := ""
	oidcNonce := ""
	authorizeRedirectURI := ""
	if params != nil && params.State != nil {
		authorizeState, _ := g.MemoryStoreProvider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			authorizeStateSplit := strings.Split(authorizeState, "@@")
			if len(authorizeStateSplit) > 1 {
				code = authorizeStateSplit[0]
				codeChallenge = authorizeStateSplit[1]
				if len(authorizeStateSplit) > 2 {
					oidcNonce = authorizeStateSplit[2]
				}
				if len(authorizeStateSplit) > 3 {
					authorizeRedirectURI = authorizeStateSplit[3]
				}
			}
			g.MemoryStoreProvider.RemoveState(refs.StringValue(params.State))
		}
	}

	nonce := uuid.New().String()
	hostname := parsers.GetHost(gc)
	authToken, err := g.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
		User:        user,
		Nonce:       nonce,
		OIDCNonce:   oidcNonce,
		Code:        code,
		Roles:       claimRoles,
		Scope:       scope,
		LoginMethod: claims.LoginMethod,
		HostName:    hostname,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to CreateAuthToken")
		return nil, err
	}

	// Store the authorization code state so /oauth/token can find it.
	// The authorizeRedirectURI is already URL-encoded from the authorize state.
	if code != "" {
		if err := g.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash+"@@"+oidcNonce+"@@"+authorizeRedirectURI); err != nil {
			log.Debug().Err(err).Msg("Failed to set code state")
			return nil, err
		}
	}

	// rollover the session for security
	sessionKey := userID
	if claims.LoginMethod != "" {
		sessionKey = claims.LoginMethod + ":" + userID
	}
	go g.MemoryStoreProvider.DeleteUserSession(sessionKey, claims.Nonce)

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

	cookie.SetSession(gc, authToken.FingerPrintHash, g.Config.AppCookieSecure)
	g.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	g.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		g.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}
	return res, nil
}

package service

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/token"
	"github.com/authorizerdev/authorizer/internal/utils"
)

// Session returns the AuthResponse bound to the caller's cookie/bearer and
// rotates the session token in the process. Transport-agnostic port of
// graphqlProvider.Session.
//
// Security note: SessionResponse carries access_token, refresh_token,
// id_token, and authenticator-enrolment fields. Per security audit C1, the
// proto annotation on Session is `mcp_tool.exposed = false` — so this
// response shape never lands in an MCP/LLM transcript.
func (p *provider) Session(ctx context.Context, meta RequestMetadata, params *model.SessionQueryRequest) (*model.AuthResponse, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Session").Logger()
	side := &ResponseSideEffects{}

	gc := &gin.Context{Request: meta.Request}
	sessionToken, err := cookie.GetSession(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get session token")
		return nil, nil, Unauthenticated("unauthorized")
	}

	claims, err := p.TokenProvider.ValidateBrowserSession(gc, sessionToken)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to validate session token")
		return nil, nil, Unauthenticated("unauthorized")
	}
	userID := claims.Subject
	log = log.With().Str("user_id", userID).Logger()
	user, err := p.StorageProvider.GetUserByID(ctx, userID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user")
		return nil, nil, err
	}

	claimRoles := append([]string{}, claims.Roles...)
	if params != nil && len(params.Roles) > 0 {
		for _, v := range params.Roles {
			if !utils.StringSliceContains(claimRoles, v) {
				log.Debug().Msg("User does not have required role")
				return nil, nil, Unauthenticated("unauthorized")
			}
		}
	}

	if params != nil {
		if err := p.enforceRequiredPermissions(ctx, log, metrics.RequiredPermissionsEndpointSession, user.ID, claimRoles, params.RequiredPermissions); err != nil {
			return nil, nil, err
		}
	}

	scope := []string{"openid", "email", "profile"}
	if params != nil && len(params.Scope) > 0 {
		scope = params.Scope
	}

	// OIDC authorize flow: if state is provided, consume the authorize state
	// and prepare code/challenge data so the authorization code can be stored
	// after token creation.
	code := ""
	codeChallenge := ""
	oidcNonce := ""
	authorizeRedirectURI := ""
	if params != nil && params.State != nil {
		authorizeState, _ := p.MemoryStoreProvider.GetState(refs.StringValue(params.State))
		if authorizeState != "" {
			parts := strings.Split(authorizeState, "@@")
			if len(parts) > 1 {
				code = parts[0]
				codeChallenge = parts[1]
				if len(parts) > 2 {
					oidcNonce = parts[2]
				}
				if len(parts) > 3 {
					authorizeRedirectURI = parts[3]
				}
			}
			p.MemoryStoreProvider.RemoveState(refs.StringValue(params.State))
		}
	}

	nonce := uuid.New().String()
	hostname := meta.HostURL
	authToken, err := p.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
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
		return nil, nil, err
	}

	if code != "" {
		if err := p.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash+"@@"+oidcNonce+"@@"+authorizeRedirectURI); err != nil {
			log.Debug().Err(err).Msg("Failed to set code state")
			return nil, nil, err
		}
	}

	sessionKey := userID
	if claims.LoginMethod != "" {
		sessionKey = claims.LoginMethod + ":" + userID
	}

	expiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if expiresIn <= 0 {
		expiresIn = 1
	}

	res := &model.AuthResponse{
		Message:     "Session token refreshed",
		AccessToken: &authToken.AccessToken.Token,
		ExpiresIn:   &expiresIn,
		IDToken:     &authToken.IDToken.Token,
		User:        user.AsAPIUser(),
	}

	// Establish the new session first, then revoke the old one. Doing both
	// synchronously closes the window where a stolen pre-rotation token
	// remains valid alongside the rotated one; doing "new then old" avoids
	// any moment where the user has no valid session token. DeleteUserSession
	// is in-memory or a single Redis DEL — failure is non-fatal.
	for _, c := range cookie.BuildSessionCookies(meta.HostURL, authToken.FingerPrintHash, p.Config.AppCookieSecure, cookie.ParseSameSite(p.Config.AppCookieSameSite)) {
		side.AddCookie(c)
	}
	p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}

	if err := p.MemoryStoreProvider.DeleteUserSession(sessionKey, claims.Nonce); err != nil {
		log.Warn().Err(err).Str("session_key", sessionKey).Msg("failed to delete old session during rollover")
	}
	return res, side, nil
}

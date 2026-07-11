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
	"github.com/authorizerdev/authorizer/internal/refs"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
	"github.com/authorizerdev/authorizer/internal/token"
)

// issueAuthResponse is the shared token-issuance tail used by every flow that
// hands an already-verified user a set of tokens (verify_otp, webauthn login,
// …). It creates the auth token, resolves and rewrites any in-flight OAuth
// authorize state (PKCE code/challenge), sets session cookies via side-effects,
// records the user session in the memory store, and fires the login/signup
// webhooks. Callers remain responsible for their own audit-log entry, which is
// flow-specific.
func (p *provider) issueAuthResponse(ctx context.Context, meta RequestMetadata, side *ResponseSideEffects, user *schemas.User, loginMethod, message string, state *string, isSignUp bool) (*model.AuthResponse, error) {
	log := p.Log.With().Str("func", "issueAuthResponse").Logger()
	// TokenProvider.CreateAuthToken takes *gin.Context but doesn't read from it;
	// reuse the request-wrapping shim so the call works for every transport.
	gc := &gin.Context{Request: meta.Request}

	roles := strings.Split(user.Roles, ",")
	scope := []string{"openid", "email", "profile"}
	code := ""
	codeChallenge := ""
	nonce := ""
	oidcNonce := ""
	authorizeRedirectURI := ""
	if state != nil {
		authorizeState, _ := p.MemoryStoreProvider.GetState(refs.StringValue(state))
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
			} else {
				nonce = authorizeState
			}
			_ = p.MemoryStoreProvider.RemoveState(refs.StringValue(state))
		}
	}
	if nonce == "" {
		nonce = uuid.New().String()
	}
	hostname := meta.HostURL
	authToken, err := p.TokenProvider.CreateAuthToken(gc, &token.AuthTokenConfig{
		User:        user,
		Roles:       roles,
		Scope:       scope,
		LoginMethod: loginMethod,
		Nonce:       nonce,
		OIDCNonce:   oidcNonce,
		Code:        code,
		HostName:    hostname,
	})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create auth token")
		return nil, err
	}

	// Code challenge could be optional if PKCE flow is not used
	if code != "" {
		if err := p.MemoryStoreProvider.SetState(code, codeChallenge+"@@"+authToken.FingerPrintHash+"@@"+oidcNonce+"@@"+authorizeRedirectURI); err != nil {
			log.Debug().Err(err).Msg("Failed to set state")
			return nil, err
		}
	}

	go func() {
		ctx := context.WithoutCancel(ctx)
		if isSignUp {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserSignUpWebhookEvent, loginMethod, user)
			// User is also logged in with signup
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		} else {
			_ = p.EventsProvider.RegisterEvent(ctx, constants.UserLoginWebhookEvent, loginMethod, user)
		}

		if err := p.StorageProvider.AddSession(ctx, &schemas.Session{
			UserID:    user.ID,
			UserAgent: meta.UserAgent,
			IP:        meta.IPAddress,
		}); err != nil {
			log.Debug().Err(err).Msg("Failed to add session")
		}
	}()

	authTokenExpiresIn := authToken.AccessToken.ExpiresAt - time.Now().Unix()
	if authTokenExpiresIn <= 0 {
		authTokenExpiresIn = 1
	}

	res := &model.AuthResponse{
		Message:     message,
		AccessToken: &authToken.AccessToken.Token,
		IDToken:     &authToken.IDToken.Token,
		ExpiresIn:   &authTokenExpiresIn,
		User:        user.AsAPIUser(),
	}

	sessionKey := loginMethod + ":" + user.ID
	for _, c := range cookie.BuildSessionCookies(hostname, authToken.FingerPrintHash, p.Config.AppCookieSecure, cookie.ParseSameSite(p.Config.AppCookieSameSite)) {
		side.AddCookie(c)
	}
	_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeSessionToken+"_"+authToken.FingerPrint, authToken.FingerPrintHash, authToken.SessionTokenExpiresAt)
	_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeAccessToken+"_"+authToken.FingerPrint, authToken.AccessToken.Token, authToken.AccessToken.ExpiresAt)

	if authToken.RefreshToken != nil {
		res.RefreshToken = &authToken.RefreshToken.Token
		_ = p.MemoryStoreProvider.SetUserSession(sessionKey, constants.TokenTypeRefreshToken+"_"+authToken.FingerPrint, authToken.RefreshToken.Token, authToken.RefreshToken.ExpiresAt)
	}
	return res, nil
}

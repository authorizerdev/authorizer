package service

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
)

// Logout ends the caller's current session: drops the memory-store session
// entry, emits expired Set-Cookie headers, records audit + metrics events.
// Transport-agnostic port of graphqlProvider.Logout.
//
// Permissions: authenticated user.
func (p *provider) Logout(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "Logout").Logger()
	side := &ResponseSideEffects{}

	gc := &gin.Context{Request: meta.Request}
	tokenData, err := p.TokenProvider.GetUserIDFromSessionOrAccessToken(gc)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get user id from session or access token")
		// No valid session/bearer -> 401, not 500. Preserve the underlying
		// message while classifying the failure as an auth error.
		return nil, nil, Unauthenticated(err.Error())
	}

	sessionKey := tokenData.UserID
	if tokenData.LoginMethod != "" {
		sessionKey = tokenData.LoginMethod + ":" + tokenData.UserID
	}
	if err := p.MemoryStoreProvider.DeleteUserSession(sessionKey, tokenData.Nonce); err != nil {
		log.Debug().Err(err).Msg("Failed to delete user session")
		return nil, nil, err
	}

	for _, c := range cookie.BuildDeleteSessionCookies(meta.HostURL, p.Config.AppCookieSecure, cookie.ParseSameSite(p.Config.AppCookieSameSite)) {
		side.AddCookie(c)
	}

	metrics.RecordAuthEvent(metrics.EventLogout, metrics.StatusSuccess)
	metrics.ActiveSessions.Dec()
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditLogoutEvent,
		Protocol: meta.Protocol, ActorID: tokenData.UserID,
		ActorType:    constants.AuditActorTypeUser,
		ResourceType: constants.AuditResourceTypeSession,
		ResourceID:   tokenData.UserID,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})

	return &model.Response{Message: "Logged out successfully"}, side, nil
}

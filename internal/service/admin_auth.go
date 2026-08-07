package service

import (
	"context"
	"fmt"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
	"github.com/gin-gonic/gin"
)

// AdminLogin validates the admin secret (constant-time) and, on success, emits
// the admin session cookie as a response side-effect. It is the only admin
// operation that does not require an existing super-admin session — it
// establishes one. Logic migrated from internal/graphql/admin_login.go.
func (p *provider) AdminLogin(ctx context.Context, meta RequestMetadata, params *model.AdminLoginRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "AdminLogin").Logger()
	// Throttled, constant-time comparison. Both this and the
	// x-authorizer-admin-secret header path share one budget so neither can be
	// brute-forced while the other is limited.
	valid, locked := p.TokenProvider.VerifyAdminSecret(meta.IPAddress, params.AdminSecret)
	if locked {
		log.Warn().Str("ip", meta.IPAddress).Msg("Admin login locked: too many failed attempts")
		p.AuditProvider.LogEvent(audit.Event{
			Action:   constants.AuditAdminLoginFailedEvent,
			Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
			ResourceType: constants.AuditResourceTypeAdminSession,
			IPAddress:    meta.IPAddress,
			UserAgent:    meta.UserAgent,
		})
		return nil, nil, TooManyRequests("too many failed attempts, please try again later")
	}
	if !valid {
		log.Debug().Msg("Invalid admin secret")
		metrics.RecordAuthEvent(metrics.EventAdminLogin, metrics.StatusFailure)
		metrics.RecordSecurityEvent("invalid_admin_secret", "admin_login")
		p.AuditProvider.LogEvent(audit.Event{
			Action:   constants.AuditAdminLoginFailedEvent,
			Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
			ResourceType: constants.AuditResourceTypeAdminSession,
			IPAddress:    meta.IPAddress,
			UserAgent:    meta.UserAgent,
		})
		return nil, nil, Unauthenticated("invalid admin secret")
	}

	// A server-side session handle, not a hash of the secret: the old cookie was
	// a stateless bearer credential with no expiry and no revocation path, so a
	// captured copy worked until the operator rotated AdminSecret — which killed
	// every admin session at once. See token/admin_session.go.
	sessionID, err := p.TokenProvider.NewAdminSession()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to create admin session")
		return nil, nil, err
	}
	side := &ResponseSideEffects{}
	side.AddCookie(cookie.BuildAdminCookie(meta.HostURL, sessionID, p.Config.AdminCookieSecure))

	metrics.RecordAuthEvent(metrics.EventAdminLogin, metrics.StatusSuccess)
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminLoginSuccessEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeAdminSession,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.Response{Message: "admin logged in successfully"}, side, nil
}

// AdminLogout clears the admin session cookie. Requires super-admin auth.
// Logic migrated from internal/graphql/admin_logout.go.
func (p *provider) AdminLogout(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error) {
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	// Actually end the session server-side. Clearing the cookie only asks THIS
	// browser to forget its copy; it does nothing about a copy an attacker
	// already exfiltrated, which is the case logout exists for.
	if sessionID, err := p.adminSessionID(meta); err == nil && sessionID != "" {
		if err := p.TokenProvider.RevokeAdminSession(sessionID); err != nil {
			p.Log.Debug().Err(err).Msg("Failed to revoke admin session")
		}
	}
	side := &ResponseSideEffects{}
	side.AddCookie(cookie.BuildDeleteAdminCookie(meta.HostURL, p.Config.AdminCookieSecure))

	metrics.RecordAuthEvent(metrics.EventAdminLogout, metrics.StatusSuccess)
	p.AuditProvider.LogEvent(audit.Event{
		Action:   constants.AuditAdminLogoutEvent,
		Protocol: meta.Protocol, ActorType: constants.AuditActorTypeAdmin,
		ResourceType: constants.AuditResourceTypeAdminSession,
		IPAddress:    meta.IPAddress,
		UserAgent:    meta.UserAgent,
	})
	return &model.Response{Message: "admin logged out successfully"}, side, nil
}

// AdminSession refreshes the admin session cookie. Requires super-admin auth.
// Logic migrated from internal/graphql/admin_session.go.
func (p *provider) AdminSession(ctx context.Context, meta RequestMetadata) (*model.Response, *ResponseSideEffects, error) {
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	// Extend the live session rather than minting a new one: the caller already
	// proved they hold a valid handle to get past requireSuperAdmin, and reusing
	// it keeps "refresh" from silently multiplying sessions.
	//
	// A caller with no cookie session still reaches here legitimately — the
	// gRPC/REST transports authenticate super-admins with the
	// x-authorizer-admin-secret header and carry no admin cookie at all. They
	// get a freshly minted session, which is what this endpoint did for every
	// caller before sessions became server-side. Refusing them would have been a
	// silent break of both transports.
	sessionID, err := p.adminSessionID(meta)
	if err != nil || sessionID == "" || p.TokenProvider.RefreshAdminSession(sessionID) != nil {
		sessionID, err = p.TokenProvider.NewAdminSession()
		if err != nil {
			p.Log.Debug().Err(err).Msg("Failed to create admin session")
			return nil, nil, err
		}
	}
	side := &ResponseSideEffects{}
	side.AddCookie(cookie.BuildAdminCookie(meta.HostURL, sessionID, p.Config.AdminCookieSecure))
	return &model.Response{Message: "admin session refreshed successfully"}, side, nil
}

// AdminMeta returns admin-only configuration metadata — configured roles,
// default roles, and protected roles. Requires super-admin auth. Logic migrated
// from internal/graphql/admin_meta.go. The schema fields are non-null lists, so
// nil slices are normalized to empty slices.
func (p *provider) AdminMeta(ctx context.Context, meta RequestMetadata) (*model.AdminMeta, *ResponseSideEffects, error) {
	if err := p.requireSuperAdmin(ctx, meta); err != nil {
		return nil, nil, err
	}
	roles := p.Config.Roles
	if roles == nil {
		roles = []string{}
	}
	defaultRoles := p.Config.DefaultRoles
	if defaultRoles == nil {
		defaultRoles = []string{}
	}
	protectedRoles := p.Config.ProtectedRoles
	if protectedRoles == nil {
		protectedRoles = []string{}
	}
	return &model.AdminMeta{
		Roles:                           roles,
		DefaultRoles:                    defaultRoles,
		ProtectedRoles:                  protectedRoles,
		IsMultiFactorAuthServiceEnabled: p.isMFAServiceAvailable(),
	}, nil, nil
}

// adminSessionID reads the caller's admin session handle off the request.
// Transport-agnostic: RequestMetadata carries the raw request for both the gin
// and gRPC paths, mirroring how ResetPassword reads the MFA cookie.
func (p *provider) adminSessionID(meta RequestMetadata) (string, error) {
	if meta.Request == nil {
		return "", fmt.Errorf("no request")
	}
	return cookie.GetAdminCookie(&gin.Context{Request: meta.Request})
}

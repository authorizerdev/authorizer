package service

import (
	"context"
	"crypto/subtle"

	"github.com/authorizerdev/authorizer/internal/audit"
	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/cookie"
	"github.com/authorizerdev/authorizer/internal/crypto"
	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/metrics"
)

// AdminLogin validates the admin secret (constant-time) and, on success, emits
// the admin session cookie as a response side-effect. It is the only admin
// operation that does not require an existing super-admin session — it
// establishes one. Logic migrated from internal/graphql/admin_login.go.
func (p *provider) AdminLogin(ctx context.Context, meta RequestMetadata, params *model.AdminLoginRequest) (*model.Response, *ResponseSideEffects, error) {
	log := p.Log.With().Str("func", "AdminLogin").Logger()
	if subtle.ConstantTimeCompare([]byte(params.AdminSecret), []byte(p.Config.AdminSecret)) != 1 {
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

	hashedKey, err := crypto.EncryptPassword(p.Config.AdminSecret)
	if err != nil {
		return nil, nil, err
	}
	side := &ResponseSideEffects{}
	side.AddCookie(cookie.BuildAdminCookie(meta.HostURL, hashedKey, p.Config.AdminCookieSecure))

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
	hashedKey, err := crypto.EncryptPassword(p.Config.AdminSecret)
	if err != nil {
		return nil, nil, err
	}
	side := &ResponseSideEffects{}
	side.AddCookie(cookie.BuildAdminCookie(meta.HostURL, hashedKey, p.Config.AdminCookieSecure))
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

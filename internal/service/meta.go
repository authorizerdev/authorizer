package service

import (
	"context"

	"github.com/authorizerdev/authorizer/internal/constants"
	"github.com/authorizerdev/authorizer/internal/graph/model"
)

// Meta returns the server's feature-flag and provider configuration.
// Stateless; no auth required; no side effects.
func (p *provider) Meta(ctx context.Context, meta RequestMetadata) (*model.Meta, *ResponseSideEffects, error) {
	c := p.Config
	return &model.Meta{
		Version:                            constants.VERSION,
		ClientID:                           c.ClientID,
		IsGoogleLoginEnabled:               c.GoogleClientID != "" && c.GoogleClientSecret != "",
		IsGithubLoginEnabled:               c.GithubClientID != "" && c.GithubClientSecret != "",
		IsFacebookLoginEnabled:             c.FacebookClientID != "" && c.FacebookClientSecret != "",
		IsLinkedinLoginEnabled:             c.LinkedinClientID != "" && c.LinkedinClientSecret != "",
		IsAppleLoginEnabled:                c.AppleClientID != "" && c.AppleClientSecret != "",
		IsTwitterLoginEnabled:              c.TwitterClientID != "" && c.TwitterClientSecret != "",
		IsMicrosoftLoginEnabled:            c.MicrosoftClientID != "" && c.MicrosoftClientSecret != "",
		IsTwitchLoginEnabled:               c.TwitchClientID != "" && c.TwitchClientSecret != "",
		IsRobloxLoginEnabled:               c.RobloxClientID != "" && c.RobloxClientSecret != "",
		IsBasicAuthenticationEnabled:       c.EnableBasicAuthentication,
		IsEmailVerificationEnabled:         c.EnableEmailVerification,
		IsMagicLinkLoginEnabled:            c.EnableMagicLinkLogin,
		IsSignUpEnabled:                    c.EnableSignup,
		IsStrongPasswordEnabled:            c.EnableStrongPassword,
		IsMultiFactorAuthEnabled:           c.EnableMFA,
		IsMobileBasicAuthenticationEnabled: c.EnableMobileBasicAuthentication,
		IsPhoneVerificationEnabled:         c.EnablePhoneVerification,
		IsOrgDiscoveryEnabled:              c.EnableOrgDiscovery,
		IsTotpMfaEnabled:                   c.EnableMFA && c.EnableTOTPLogin,
		IsEmailOtpMfaEnabled:               c.EnableMFA && c.EnableEmailOTP && c.IsEmailServiceEnabled,
		IsSmsOtpMfaEnabled:                 c.EnableMFA && c.EnableSMSOTP && c.IsSMSServiceEnabled,
		// WebAuthn/passkey as an MFA factor is gated by --disable-webauthn-mfa
		// (EnableWebauthnMFA), same shape as the other per-method flags. Does
		// not affect WebAuthn/passkey as a primary login method.
		IsWebauthnEnabled: c.EnableMFA && c.EnableWebauthnMFA,
		IsMfaEnforced:     c.EnforceMFA,
	}, nil, nil
}
